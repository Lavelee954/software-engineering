package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/system-trading/core/internal/agents"
	"github.com/system-trading/core/internal/infrastructure/brokers"
	"github.com/system-trading/core/internal/infrastructure/config"
	"github.com/system-trading/core/internal/infrastructure/logger"
	"github.com/system-trading/core/internal/infrastructure/messagebus"
	"github.com/system-trading/core/internal/infrastructure/metrics"
	"github.com/system-trading/core/internal/usecases"
	"github.com/system-trading/core/internal/usecases/interfaces"
)

type Application struct {
	config        *config.Config
	logger        *logger.ZapLogger
	metrics       *metrics.PrometheusMetrics
	messageBus    *messagebus.NATSBus
	
	orderService     *usecases.OrderService
	portfolioService *usecases.PortfolioService
	riskService      *usecases.RiskService
	executionAgent   *agents.ExecutionAgent
	
	httpServer    *http.Server
}

func main() {
	if err := run(); err != nil {
		log.Fatalf("Application failed: %v", err)
	}
}

func run() error {
	app, err := initializeApplication()
	if err != nil {
		return fmt.Errorf("failed to initialize application: %w", err)
	}

	if err := app.Start(); err != nil {
		return fmt.Errorf("failed to start application: %w", err)
	}

	defer app.Shutdown()

	app.logger.Info("System Trading Core started successfully",
		interfaces.Field{Key: "version", Value: "1.0.0"},
		interfaces.Field{Key: "environment", Value: getEnv("ENVIRONMENT", "development")},
	)

	return app.WaitForShutdown()
}

func initializeApplication() (*Application, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	appLogger, err := logger.NewZapLogger(cfg.Logging)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	appMetrics := metrics.NewPrometheusMetrics("system-trading-core")

	busConfig := messagebus.Config{
		URL:               cfg.NATS.URL,
		MaxReconnects:     cfg.NATS.MaxReconnects,
		ReconnectWait:     cfg.NATS.ReconnectWait,
		ConnectionTimeout: cfg.NATS.ConnectionTimeout,
		DrainTimeout:      cfg.NATS.DrainTimeout,
	}

	bus, err := messagebus.NewNATSBus(busConfig, appLogger, appMetrics)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize message bus: %w", err)
	}

	app := &Application{
		config:     cfg,
		logger:     appLogger,
		metrics:    appMetrics,
		messageBus: bus,
	}

	if err := app.initializeServices(); err != nil {
		return nil, fmt.Errorf("failed to initialize services: %w", err)
	}

	if err := app.setupHTTPServer(); err != nil {
		return nil, fmt.Errorf("failed to setup HTTP server: %w", err)
	}

	return app, nil
}

func (app *Application) initializeServices() error {
	riskLimits := &interfaces.RiskLimits{
		MaxPositionSize:    app.config.Risk.MaxPositionSize,
		MaxConcentration:   app.config.Risk.MaxConcentration,
		MaxLeverage:        app.config.Risk.MaxLeverage,
		MaxDailyLoss:       app.config.Risk.MaxDailyLoss,
		MaxVaR:             app.config.Risk.MaxVaR,
		VaRConfidenceLevel: app.config.Risk.VaRConfidenceLevel,
	}

	app.portfolioService = usecases.NewPortfolioService(
		nil, // TODO: Implement repository
		app.messageBus,
		app.logger,
		app.metrics,
	)

	app.orderService = usecases.NewOrderService(
		nil, // TODO: Implement repository
		app.messageBus,
		app.logger,
		app.metrics,
		nil, // TODO: Implement validator
	)

	app.riskService = usecases.NewRiskService(
		app.portfolioService,
		app.messageBus,
		app.logger,
		app.metrics,
		riskLimits,
	)

	// Initialize Execution Agent with Mock Broker
	trader := brokers.NewMockBroker("MockBroker", app.logger)
	app.executionAgent = agents.NewExecutionAgent(
		app.messageBus,
		trader,
		app.logger,
		app.metrics,
	)

	return nil
}

func (app *Application) setupHTTPServer() error {
	mux := http.NewServeMux()

	mux.Handle("/metrics", promhttp.Handler())

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status": "healthy", "timestamp": "%s"}`, time.Now().Format(time.RFC3339))
	})

	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		if !app.messageBus.IsConnected() {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, `{"status": "not ready", "reason": "message bus disconnected"}`)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status": "ready", "timestamp": "%s"}`, time.Now().Format(time.RFC3339))
	})

	serverAddr := fmt.Sprintf("%s:%d", app.config.Server.Host, app.config.Server.Port)
	
	app.httpServer = &http.Server{
		Addr:         serverAddr,
		Handler:      mux,
		ReadTimeout:  app.config.Server.ReadTimeout,
		WriteTimeout: app.config.Server.WriteTimeout,
		IdleTimeout:  app.config.Server.IdleTimeout,
	}

	return nil
}

func (app *Application) Start() error {
	// Start Execution Agent
	ctx := context.Background()
	if err := app.executionAgent.Start(ctx); err != nil {
		return fmt.Errorf("failed to start execution agent: %w", err)
	}

	go func() {
		app.logger.Info("Starting HTTP server",
			interfaces.Field{Key: "addr", Value: app.httpServer.Addr},
		)
		
		if err := app.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			app.logger.Error("HTTP server failed",
				interfaces.Field{Key: "error", Value: err},
			)
		}
	}()

	if err := app.subscribeToMessageBusTopics(); err != nil {
		return fmt.Errorf("failed to subscribe to message bus topics: %w", err)
	}

	return nil
}

func (app *Application) subscribeToMessageBusTopics() error {
	ctx := context.Background()

	if err := app.messageBus.Subscribe(ctx, "order.executed", app.handleOrderExecuted); err != nil {
		return fmt.Errorf("failed to subscribe to order.executed: %w", err)
	}

	if err := app.messageBus.Subscribe(ctx, "order.proposed", app.handleOrderProposed); err != nil {
		return fmt.Errorf("failed to subscribe to order.proposed: %w", err)
	}

	app.logger.Info("Subscribed to message bus topics")
	return nil
}

func (app *Application) handleOrderExecuted(ctx context.Context, message []byte) error {
	return nil
}

func (app *Application) handleOrderProposed(ctx context.Context, message []byte) error {
	return nil
}

func (app *Application) WaitForShutdown() error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	sig := <-quit
	app.logger.Info("Received shutdown signal",
		interfaces.Field{Key: "signal", Value: sig.String()},
	)

	return nil
}

func (app *Application) Shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	app.logger.Info("Shutting down application")

	// Stop execution agent first
	if app.executionAgent != nil {
		if err := app.executionAgent.Stop(ctx); err != nil {
			app.logger.Error("Execution agent shutdown failed",
				interfaces.Field{Key: "error", Value: err},
			)
		} else {
			app.logger.Info("Execution agent shut down gracefully")
		}
	}

	if app.httpServer != nil {
		if err := app.httpServer.Shutdown(ctx); err != nil {
			app.logger.Error("HTTP server shutdown failed",
				interfaces.Field{Key: "error", Value: err},
			)
		} else {
			app.logger.Info("HTTP server shut down gracefully")
		}
	}

	if app.messageBus != nil {
		if err := app.messageBus.Close(); err != nil {
			app.logger.Error("Message bus close failed",
				interfaces.Field{Key: "error", Value: err},
			)
		} else {
			app.logger.Info("Message bus closed gracefully")
		}
	}

	if app.logger != nil {
		app.logger.Sync()
	}

	app.logger.Info("Application shutdown complete")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}