package agents

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/system-trading/core/internal/entities"
	"github.com/system-trading/core/internal/usecases/interfaces"
)

type DataCollectorAgent struct {
	messageBus      interfaces.MessageBus
	priceProvider   interfaces.PriceProvider
	newsProvider    interfaces.NewsProvider
	marketDataRepo  interfaces.MarketDataRepository
	logger          interfaces.Logger
	metrics         interfaces.MetricsCollector
	
	subscriptions   map[entities.Symbol]bool
	mu              sync.RWMutex
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
}

type DataCollectorConfig struct {
	SubscriptionSymbols []entities.Symbol `json:"symbols"`
	NewsUpdateInterval  time.Duration     `json:"news_interval"`
	HealthCheckInterval time.Duration     `json:"health_interval"`
}

func NewDataCollectorAgent(
	messageBus interfaces.MessageBus,
	priceProvider interfaces.PriceProvider,
	newsProvider interfaces.NewsProvider,
	marketDataRepo interfaces.MarketDataRepository,
	logger interfaces.Logger,
	metrics interfaces.MetricsCollector,
	config DataCollectorConfig,
) *DataCollectorAgent {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &DataCollectorAgent{
		messageBus:     messageBus,
		priceProvider:  priceProvider,
		newsProvider:   newsProvider,
		marketDataRepo: marketDataRepo,
		logger:         logger,
		metrics:        metrics,
		subscriptions:  make(map[entities.Symbol]bool),
		ctx:            ctx,
		cancel:         cancel,
	}
}

func (a *DataCollectorAgent) Start(config DataCollectorConfig) error {
	a.logger.Info("Starting Data Collector Agent",
		interfaces.Field{Key: "symbols", Value: len(config.SubscriptionSymbols)},
	)

	if err := a.subscribeToMarketData(config.SubscriptionSymbols); err != nil {
		return fmt.Errorf("failed to subscribe to market data: %w", err)
	}

	a.wg.Add(3)
	go a.runNewsCollector(config.NewsUpdateInterval)
	go a.runHealthMonitor(config.HealthCheckInterval)
	go a.runMacroDataCollector()

	a.metrics.SetGauge("agent_health", 1, map[string]string{
		"agent_name": "data_collector",
	})

	a.logger.Info("Data Collector Agent started successfully")
	return nil
}

func (a *DataCollectorAgent) Stop() error {
	a.logger.Info("Stopping Data Collector Agent")

	a.cancel()

	a.mu.Lock()
	for symbol := range a.subscriptions {
		if err := a.priceProvider.UnsubscribeFromPrice(a.ctx, symbol); err != nil {
			a.logger.Warn("Failed to unsubscribe from price data",
				interfaces.Field{Key: "symbol", Value: symbol},
				interfaces.Field{Key: "error", Value: err},
			)
		}
	}
	a.subscriptions = make(map[entities.Symbol]bool)
	a.mu.Unlock()

	done := make(chan struct{})
	go func() {
		a.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		a.logger.Info("Data Collector Agent stopped gracefully")
	case <-time.After(10 * time.Second):
		a.logger.Warn("Data Collector Agent stop timeout")
	}

	a.metrics.SetGauge("agent_health", 0, map[string]string{
		"agent_name": "data_collector",
	})

	return nil
}

func (a *DataCollectorAgent) AddSymbol(symbol entities.Symbol) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.subscriptions[symbol] {
		return fmt.Errorf("already subscribed to symbol: %s", symbol)
	}

	if err := a.priceProvider.SubscribeToPrice(a.ctx, symbol, a.handlePriceUpdate); err != nil {
		return fmt.Errorf("failed to subscribe to price for %s: %w", symbol, err)
	}

	a.subscriptions[symbol] = true

	a.logger.Info("Added symbol subscription",
		interfaces.Field{Key: "symbol", Value: symbol},
	)

	return nil
}

func (a *DataCollectorAgent) RemoveSymbol(symbol entities.Symbol) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.subscriptions[symbol] {
		return fmt.Errorf("not subscribed to symbol: %s", symbol)
	}

	if err := a.priceProvider.UnsubscribeFromPrice(a.ctx, symbol); err != nil {
		return fmt.Errorf("failed to unsubscribe from price for %s: %w", symbol, err)
	}

	delete(a.subscriptions, symbol)

	a.logger.Info("Removed symbol subscription",
		interfaces.Field{Key: "symbol", Value: symbol},
	)

	return nil
}

func (a *DataCollectorAgent) subscribeToMarketData(symbols []entities.Symbol) error {
	for _, symbol := range symbols {
		if err := a.AddSymbol(symbol); err != nil {
			a.logger.Error("Failed to subscribe to symbol",
				interfaces.Field{Key: "symbol", Value: symbol},
				interfaces.Field{Key: "error", Value: err},
			)
			continue
		}
	}
	return nil
}

func (a *DataCollectorAgent) handlePriceUpdate(marketData *entities.MarketData) {
	start := time.Now()
	defer func() {
		a.metrics.RecordDuration("market_data_processing_duration", time.Since(start).Seconds(), map[string]string{
			"symbol": string(marketData.Symbol),
		})
	}()

	latency := time.Since(marketData.Timestamp)
	a.metrics.RecordDuration("market_data_latency", latency.Seconds(), map[string]string{
		"symbol":    string(marketData.Symbol),
		"data_type": "price",
	})

	if err := a.marketDataRepo.SaveMarketData(a.ctx, marketData); err != nil {
		a.logger.Error("Failed to save market data",
			interfaces.Field{Key: "symbol", Value: marketData.Symbol},
			interfaces.Field{Key: "error", Value: err},
		)
		a.metrics.IncrementCounter("market_data_save_errors", map[string]string{
			"symbol": string(marketData.Symbol),
		})
		return
	}

	if err := a.messageBus.Publish(a.ctx, "raw.market_data", marketData); err != nil {
		a.logger.Error("Failed to publish market data",
			interfaces.Field{Key: "symbol", Value: marketData.Symbol},
			interfaces.Field{Key: "error", Value: err},
		)
		a.metrics.IncrementCounter("market_data_publish_errors", map[string]string{
			"symbol": string(marketData.Symbol),
		})
		return
	}

	a.metrics.IncrementCounter("market_data_processed", map[string]string{
		"symbol": string(marketData.Symbol),
	})

	a.logger.Debug("Market data processed and published",
		interfaces.Field{Key: "symbol", Value: marketData.Symbol},
		interfaces.Field{Key: "price", Value: marketData.Price},
		interfaces.Field{Key: "latency_ms", Value: latency.Milliseconds()},
	)
}

func (a *DataCollectorAgent) runNewsCollector(interval time.Duration) {
	defer a.wg.Done()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	a.logger.Info("Starting news collection",
		interfaces.Field{Key: "interval", Value: interval},
	)

	if err := a.newsProvider.SubscribeToNews(a.ctx, a.handleNewsUpdate); err != nil {
		a.logger.Error("Failed to subscribe to news",
			interfaces.Field{Key: "error", Value: err},
		)
		return
	}

	for {
		select {
		case <-a.ctx.Done():
			a.logger.Info("News collector stopped")
			return
		case <-ticker.C:
			a.collectLatestNews()
		}
	}
}

func (a *DataCollectorAgent) collectLatestNews() {
	a.mu.RLock()
	symbols := make([]entities.Symbol, 0, len(a.subscriptions))
	for symbol := range a.subscriptions {
		symbols = append(symbols, symbol)
	}
	a.mu.RUnlock()

	if len(symbols) == 0 {
		return
	}

	news, err := a.newsProvider.GetLatestNews(a.ctx, symbols)
	if err != nil {
		a.logger.Error("Failed to get latest news",
			interfaces.Field{Key: "error", Value: err},
		)
		a.metrics.IncrementCounter("news_collection_errors", map[string]string{
			"source": "api",
		})
		return
	}

	for _, article := range news {
		a.handleNewsUpdate(article)
	}

	a.metrics.IncrementCounter("news_collection_runs", map[string]string{
		"status": "success",
	})
}

func (a *DataCollectorAgent) handleNewsUpdate(article *entities.NewsArticle) {
	if err := a.marketDataRepo.SaveNewsArticle(a.ctx, article); err != nil {
		a.logger.Error("Failed to save news article",
			interfaces.Field{Key: "article_id", Value: article.ID},
			interfaces.Field{Key: "error", Value: err},
		)
		a.metrics.IncrementCounter("news_save_errors", map[string]string{
			"source": article.Source,
		})
		return
	}

	if err := a.messageBus.Publish(a.ctx, "raw.news.article", article); err != nil {
		a.logger.Error("Failed to publish news article",
			interfaces.Field{Key: "article_id", Value: article.ID},
			interfaces.Field{Key: "error", Value: err},
		)
		a.metrics.IncrementCounter("news_publish_errors", map[string]string{
			"source": article.Source,
		})
		return
	}

	a.metrics.IncrementCounter("news_articles_processed", map[string]string{
		"source": article.Source,
	})

	a.logger.Debug("News article processed and published",
		interfaces.Field{Key: "article_id", Value: article.ID},
		interfaces.Field{Key: "title", Value: article.Title},
		interfaces.Field{Key: "source", Value: article.Source},
	)
}

func (a *DataCollectorAgent) runMacroDataCollector() {
	defer a.wg.Done()

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	a.logger.Info("Starting macro data collection")

	for {
		select {
		case <-a.ctx.Done():
			a.logger.Info("Macro data collector stopped")
			return
		case <-ticker.C:
			a.collectMacroData()
		}
	}
}

func (a *DataCollectorAgent) collectMacroData() {
	indicators := []MacroIndicatorSource{
		{Name: "GDP", Country: "US", Impact: "HIGH"},
		{Name: "INFLATION_RATE", Country: "US", Impact: "HIGH"},
		{Name: "UNEMPLOYMENT_RATE", Country: "US", Impact: "MEDIUM"},
		{Name: "INTEREST_RATE", Country: "US", Impact: "HIGH"},
	}

	for _, source := range indicators {
		indicator := &entities.MacroIndicator{
			Name:      source.Name,
			Value:     a.fetchMacroValue(source.Name),
			Country:   source.Country,
			Period:    "MONTHLY",
			Timestamp: time.Now(),
			Impact:    source.Impact,
		}

		if err := a.marketDataRepo.SaveMacroIndicator(a.ctx, indicator); err != nil {
			a.logger.Error("Failed to save macro indicator",
				interfaces.Field{Key: "indicator", Value: source.Name},
				interfaces.Field{Key: "error", Value: err},
			)
			continue
		}

		if err := a.messageBus.Publish(a.ctx, "raw.macro.indicator", indicator); err != nil {
			a.logger.Error("Failed to publish macro indicator",
				interfaces.Field{Key: "indicator", Value: source.Name},
				interfaces.Field{Key: "error", Value: err},
			)
			continue
		}

		a.logger.Debug("Macro indicator processed",
			interfaces.Field{Key: "indicator", Value: source.Name},
			interfaces.Field{Key: "value", Value: indicator.Value},
		)
	}

	a.metrics.IncrementCounter("macro_data_collection_runs", map[string]string{
		"status": "success",
	})
}

func (a *DataCollectorAgent) runHealthMonitor(interval time.Duration) {
	defer a.wg.Done()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			a.publishHealthStatus()
		}
	}
}

func (a *DataCollectorAgent) publishHealthStatus() {
	a.mu.RLock()
	subscriptionCount := len(a.subscriptions)
	a.mu.RUnlock()

	healthStatus := HealthStatusMessage{
		ComponentName: "data_collector",
		Status:        "healthy",
		Metrics: map[string]interface{}{
			"active_subscriptions": subscriptionCount,
			"uptime_seconds":       time.Since(time.Now()).Seconds(),
		},
		Timestamp: time.Now(),
	}

	if err := a.messageBus.Publish(a.ctx, "system.health", healthStatus); err != nil {
		a.logger.Warn("Failed to publish health status",
			interfaces.Field{Key: "error", Value: err},
		)
	}
}

func (a *DataCollectorAgent) fetchMacroValue(indicator string) float64 {
	values := map[string]float64{
		"GDP":                2.1,
		"INFLATION_RATE":     3.2,
		"UNEMPLOYMENT_RATE":  4.1,
		"INTEREST_RATE":      5.25,
	}
	
	if value, exists := values[indicator]; exists {
		return value
	}
	return 0.0
}

type MacroIndicatorSource struct {
	Name    string
	Country string
	Impact  string
}

type HealthStatusMessage struct {
	ComponentName string                 `json:"component_name"`
	Status        string                 `json:"status"`
	Metrics       map[string]interface{} `json:"metrics"`
	Timestamp     time.Time              `json:"timestamp"`
}