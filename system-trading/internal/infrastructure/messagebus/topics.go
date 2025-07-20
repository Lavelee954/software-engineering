package messagebus

import (
	"time"

	"github.com/system-trading/core/internal/entities"
)

const (
	TopicRawMarketData   = "raw.market_data"
	TopicRawNewsArticle  = "raw.news.article"
	TopicRawMacro        = "raw.macro.indicator"
	TopicInsightTechnical = "insight.technical"
	TopicInsightSentiment = "insight.sentiment"
	TopicInsightMacro    = "insight.macro"
	TopicOrderProposed   = "order.proposed"
	TopicOrderApproved   = "order.approved"
	TopicOrderExecuted   = "order.executed"
	TopicOrderRejected   = "order.rejected"
	TopicRiskAlert       = "risk.alert"
	TopicPortfolioUpdate = "portfolio.update"
	TopicSystemHealth    = "system.health"
)

type MessageEnvelope struct {
	MessageID   string      `json:"message_id"`
	Topic       string      `json:"topic"`
	Timestamp   time.Time   `json:"timestamp"`
	Source      string      `json:"source"`
	Data        interface{} `json:"data"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type MarketDataMessage struct {
	MessageEnvelope
	Data entities.MarketData `json:"data"`
}

type NewsMessage struct {
	MessageEnvelope
	Data entities.NewsArticle `json:"data"`
}

type MacroIndicatorMessage struct {
	MessageEnvelope
	Data entities.MacroIndicator `json:"data"`
}

type TechnicalInsightMessage struct {
	MessageEnvelope
	Data entities.TechnicalIndicator `json:"data"`
}

type SentimentInsightMessage struct {
	MessageEnvelope
	Data entities.SentimentScore `json:"data"`
}

type MacroInsightMessage struct {
	MessageEnvelope
	Data MacroInsight `json:"data"`
}

type OrderMessage struct {
	MessageEnvelope
	Data entities.Order `json:"data"`
}

type PortfolioUpdateMessage struct {
	MessageEnvelope
	Data PortfolioUpdate `json:"data"`
}

type RiskAlertMessage struct {
	MessageEnvelope
	Data RiskAlert `json:"data"`
}

type HealthCheckMessage struct {
	MessageEnvelope
	Data HealthStatus `json:"data"`
}

type MacroInsight struct {
	Regime       string             `json:"regime"`
	Indicators   map[string]float64 `json:"indicators"`
	Confidence   float64            `json:"confidence"`
	Outlook      string             `json:"outlook"`
	RiskLevel    string             `json:"risk_level"`
	Timestamp    time.Time          `json:"timestamp"`
}

type PortfolioUpdate struct {
	PortfolioID   string                      `json:"portfolio_id"`
	TotalValue    float64                     `json:"total_value"`
	Cash          float64                     `json:"cash"`
	Positions     map[entities.Symbol]float64 `json:"positions"`
	TotalPnL      float64                     `json:"total_pnl"`
	DayPnL        float64                     `json:"day_pnl"`
	Timestamp     time.Time                   `json:"timestamp"`
}

type RiskAlert struct {
	AlertType     string                 `json:"alert_type"`
	Severity      string                 `json:"severity"`
	Symbol        entities.Symbol        `json:"symbol,omitempty"`
	CurrentValue  float64                `json:"current_value"`
	Threshold     float64                `json:"threshold"`
	Message       string                 `json:"message"`
	Recommendations []string             `json:"recommendations,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	Timestamp     time.Time              `json:"timestamp"`
}

type HealthStatus struct {
	ComponentName string                 `json:"component_name"`
	Status        string                 `json:"status"`
	Latency       time.Duration          `json:"latency"`
	Message       string                 `json:"message,omitempty"`
	Metrics       map[string]interface{} `json:"metrics,omitempty"`
	Timestamp     time.Time              `json:"timestamp"`
}

func NewMessageEnvelope(topic, source string, data interface{}) MessageEnvelope {
	return MessageEnvelope{
		MessageID: generateMessageID(),
		Topic:     topic,
		Timestamp: time.Now(),
		Source:    source,
		Data:      data,
		Metadata:  make(map[string]interface{}),
	}
}

func generateMessageID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(12)
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}