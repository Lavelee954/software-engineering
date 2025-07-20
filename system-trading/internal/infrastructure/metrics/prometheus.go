package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type PrometheusMetrics struct {
	// Message Bus Metrics
	messagesPublished     *prometheus.CounterVec
	messagesHandled       *prometheus.CounterVec
	messagePublishErrors  *prometheus.CounterVec
	messageHandleErrors   *prometheus.CounterVec
	publishDuration       *prometheus.HistogramVec
	handleDuration        *prometheus.HistogramVec

	// Trading Metrics
	ordersTotal           *prometheus.CounterVec
	ordersFilled          *prometheus.CounterVec
	ordersRejected        *prometheus.CounterVec
	orderFillDuration     *prometheus.HistogramVec
	tradingVolume         *prometheus.CounterVec
	portfolioValue        *prometheus.GaugeVec
	positionCount         *prometheus.GaugeVec

	// Risk Metrics
	riskAlerts            *prometheus.CounterVec
	portfolioRisk         *prometheus.GaugeVec
	positionRisk          *prometheus.GaugeVec
	varValue              *prometheus.GaugeVec

	// System Metrics
	agentHealth           *prometheus.GaugeVec
	connectionStatus      *prometheus.GaugeVec
	errorRate             *prometheus.CounterVec
	responseTime          *prometheus.HistogramVec

	// Market Data Metrics
	marketDataLatency     *prometheus.HistogramVec
	priceUpdates          *prometheus.CounterVec
	newsArticlesProcessed *prometheus.CounterVec
}

func NewPrometheusMetrics(serviceName string) *PrometheusMetrics {
	labels := prometheus.Labels{"service": serviceName}

	return &PrometheusMetrics{
		// Message Bus Metrics
		messagesPublished: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "messages_published_total",
				Help:        "Total number of messages published to message bus",
				ConstLabels: labels,
			},
			[]string{"topic"},
		),
		messagesHandled: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "messages_handled_total",
				Help:        "Total number of messages handled from message bus",
				ConstLabels: labels,
			},
			[]string{"topic"},
		),
		messagePublishErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "message_publish_errors_total",
				Help:        "Total number of message publish errors",
				ConstLabels: labels,
			},
			[]string{"topic", "error"},
		),
		messageHandleErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "message_handle_errors_total",
				Help:        "Total number of message handle errors",
				ConstLabels: labels,
			},
			[]string{"topic"},
		),
		publishDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:        "message_publish_duration_seconds",
				Help:        "Time taken to publish messages",
				ConstLabels: labels,
				Buckets:     prometheus.DefBuckets,
			},
			[]string{"topic"},
		),
		handleDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:        "message_handle_duration_seconds",
				Help:        "Time taken to handle messages",
				ConstLabels: labels,
				Buckets:     prometheus.DefBuckets,
			},
			[]string{"topic"},
		),

		// Trading Metrics
		ordersTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "orders_total",
				Help:        "Total number of orders created",
				ConstLabels: labels,
			},
			[]string{"symbol", "side", "type", "status"},
		),
		ordersFilled: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "orders_filled_total",
				Help:        "Total number of orders filled",
				ConstLabels: labels,
			},
			[]string{"symbol", "side"},
		),
		ordersRejected: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "orders_rejected_total",
				Help:        "Total number of orders rejected",
				ConstLabels: labels,
			},
			[]string{"symbol", "reason"},
		),
		orderFillDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:        "order_fill_duration_seconds",
				Help:        "Time taken to fill orders",
				ConstLabels: labels,
				Buckets:     []float64{0.1, 0.5, 1.0, 2.0, 5.0, 10.0, 30.0},
			},
			[]string{"symbol"},
		),
		tradingVolume: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "trading_volume_total",
				Help:        "Total trading volume",
				ConstLabels: labels,
			},
			[]string{"symbol", "side"},
		),
		portfolioValue: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name:        "portfolio_value",
				Help:        "Current portfolio value",
				ConstLabels: labels,
			},
			[]string{"portfolio_id"},
		),
		positionCount: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name:        "position_count",
				Help:        "Number of open positions",
				ConstLabels: labels,
			},
			[]string{"portfolio_id"},
		),

		// Risk Metrics
		riskAlerts: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "risk_alerts_total",
				Help:        "Total number of risk alerts triggered",
				ConstLabels: labels,
			},
			[]string{"alert_type", "severity"},
		),
		portfolioRisk: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name:        "portfolio_risk",
				Help:        "Current portfolio risk metrics",
				ConstLabels: labels,
			},
			[]string{"portfolio_id", "metric"},
		),
		positionRisk: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name:        "position_risk",
				Help:        "Current position risk metrics",
				ConstLabels: labels,
			},
			[]string{"symbol", "metric"},
		),
		varValue: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name:        "var_value",
				Help:        "Value at Risk",
				ConstLabels: labels,
			},
			[]string{"portfolio_id", "confidence_level"},
		),

		// System Metrics
		agentHealth: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name:        "agent_health",
				Help:        "Agent health status (1=healthy, 0=unhealthy)",
				ConstLabels: labels,
			},
			[]string{"agent_name"},
		),
		connectionStatus: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name:        "connection_status",
				Help:        "Connection status (1=connected, 0=disconnected)",
				ConstLabels: labels,
			},
			[]string{"connection_type", "target"},
		),
		errorRate: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "errors_total",
				Help:        "Total number of errors",
				ConstLabels: labels,
			},
			[]string{"component", "error_type"},
		),
		responseTime: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:        "response_time_seconds",
				Help:        "Response time for operations",
				ConstLabels: labels,
				Buckets:     prometheus.DefBuckets,
			},
			[]string{"operation"},
		),

		// Market Data Metrics
		marketDataLatency: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:        "market_data_latency_seconds",
				Help:        "Market data latency",
				ConstLabels: labels,
				Buckets:     []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0},
			},
			[]string{"symbol", "data_type"},
		),
		priceUpdates: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "price_updates_total",
				Help:        "Total number of price updates received",
				ConstLabels: labels,
			},
			[]string{"symbol"},
		),
		newsArticlesProcessed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name:        "news_articles_processed_total",
				Help:        "Total number of news articles processed",
				ConstLabels: labels,
			},
			[]string{"source"},
		),
	}
}

func (m *PrometheusMetrics) IncrementCounter(name string, labels map[string]string) {
	switch name {
	case "message_bus_published":
		m.messagesPublished.With(prometheus.Labels(labels)).Inc()
	case "message_bus_handled":
		m.messagesHandled.With(prometheus.Labels(labels)).Inc()
	case "message_bus_publish_errors":
		m.messagePublishErrors.With(prometheus.Labels(labels)).Inc()
	case "message_bus_handle_errors":
		m.messageHandleErrors.With(prometheus.Labels(labels)).Inc()
	case "orders_total":
		m.ordersTotal.With(prometheus.Labels(labels)).Inc()
	case "orders_filled":
		m.ordersFilled.With(prometheus.Labels(labels)).Inc()
	case "orders_rejected":
		m.ordersRejected.With(prometheus.Labels(labels)).Inc()
	case "trading_volume":
		m.tradingVolume.With(prometheus.Labels(labels)).Inc()
	case "risk_alerts":
		m.riskAlerts.With(prometheus.Labels(labels)).Inc()
	case "errors_total":
		m.errorRate.With(prometheus.Labels(labels)).Inc()
	case "price_updates":
		m.priceUpdates.With(prometheus.Labels(labels)).Inc()
	case "news_articles_processed":
		m.newsArticlesProcessed.With(prometheus.Labels(labels)).Inc()
	}
}

func (m *PrometheusMetrics) RecordDuration(name string, duration float64, labels map[string]string) {
	switch name {
	case "message_bus_publish_duration":
		m.publishDuration.With(prometheus.Labels(labels)).Observe(duration)
	case "message_bus_handle_duration":
		m.handleDuration.With(prometheus.Labels(labels)).Observe(duration)
	case "order_fill_duration":
		m.orderFillDuration.With(prometheus.Labels(labels)).Observe(duration)
	case "response_time":
		m.responseTime.With(prometheus.Labels(labels)).Observe(duration)
	case "market_data_latency":
		m.marketDataLatency.With(prometheus.Labels(labels)).Observe(duration)
	}
}

func (m *PrometheusMetrics) SetGauge(name string, value float64, labels map[string]string) {
	switch name {
	case "portfolio_value":
		m.portfolioValue.With(prometheus.Labels(labels)).Set(value)
	case "position_count":
		m.positionCount.With(prometheus.Labels(labels)).Set(value)
	case "portfolio_risk":
		m.portfolioRisk.With(prometheus.Labels(labels)).Set(value)
	case "position_risk":
		m.positionRisk.With(prometheus.Labels(labels)).Set(value)
	case "var_value":
		m.varValue.With(prometheus.Labels(labels)).Set(value)
	case "agent_health":
		m.agentHealth.With(prometheus.Labels(labels)).Set(value)
	case "connection_status":
		m.connectionStatus.With(prometheus.Labels(labels)).Set(value)
	}
}

func (m *PrometheusMetrics) RecordOrderMetrics(order map[string]interface{}) {
	labels := prometheus.Labels{
		"symbol": order["symbol"].(string),
		"side":   order["side"].(string),
		"type":   order["type"].(string),
		"status": order["status"].(string),
	}
	m.ordersTotal.With(labels).Inc()

	if order["status"] == "EXECUTED" {
		fillLabels := prometheus.Labels{
			"symbol": order["symbol"].(string),
			"side":   order["side"].(string),
		}
		m.ordersFilled.With(fillLabels).Inc()

		if fillTime, ok := order["fill_duration"].(time.Duration); ok {
			durationLabels := prometheus.Labels{
				"symbol": order["symbol"].(string),
			}
			m.orderFillDuration.With(durationLabels).Observe(fillTime.Seconds())
		}
	}
}

func (m *PrometheusMetrics) RecordPortfolioMetrics(portfolioID string, value float64, positionCount int) {
	portfolioLabels := prometheus.Labels{"portfolio_id": portfolioID}
	m.portfolioValue.With(portfolioLabels).Set(value)
	m.positionCount.With(portfolioLabels).Set(float64(positionCount))
}

func (m *PrometheusMetrics) RecordRiskMetrics(portfolioID string, var95, var99, leverage float64) {
	var95Labels := prometheus.Labels{
		"portfolio_id":     portfolioID,
		"confidence_level": "95",
	}
	var99Labels := prometheus.Labels{
		"portfolio_id":     portfolioID,
		"confidence_level": "99",
	}
	leverageLabels := prometheus.Labels{
		"portfolio_id": portfolioID,
		"metric":       "leverage",
	}

	m.varValue.With(var95Labels).Set(var95)
	m.varValue.With(var99Labels).Set(var99)
	m.portfolioRisk.With(leverageLabels).Set(leverage)
}

func (m *PrometheusMetrics) RecordAgentHealth(agentName string, isHealthy bool) {
	labels := prometheus.Labels{"agent_name": agentName}
	if isHealthy {
		m.agentHealth.With(labels).Set(1)
	} else {
		m.agentHealth.With(labels).Set(0)
	}
}

func (m *PrometheusMetrics) RecordConnectionStatus(connectionType, target string, isConnected bool) {
	labels := prometheus.Labels{
		"connection_type": connectionType,
		"target":          target,
	}
	if isConnected {
		m.connectionStatus.With(labels).Set(1)
	} else {
		m.connectionStatus.With(labels).Set(0)
	}
}