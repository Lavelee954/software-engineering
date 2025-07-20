package interfaces

import (
	"context"

	"github.com/system-trading/core/internal/entities"
)

type MessageBus interface {
	Publish(ctx context.Context, topic string, message interface{}) error
	Subscribe(ctx context.Context, topic string, handler MessageHandler) error
	Close() error
}

type MessageHandler func(ctx context.Context, message []byte) error

type Logger interface {
	Info(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Debug(msg string, fields ...Field)
}

type Field struct {
	Key   string
	Value interface{}
}

type MetricsCollector interface {
	IncrementCounter(name string, labels map[string]string)
	RecordDuration(name string, duration float64, labels map[string]string)
	SetGauge(name string, value float64, labels map[string]string)
}

type RiskCalculator interface {
	CalculatePositionRisk(portfolio *entities.Portfolio, order *entities.Order) (*RiskMetrics, error)
	CalculatePortfolioRisk(portfolio *entities.Portfolio) (*PortfolioRisk, error)
	ValidateOrder(portfolio *entities.Portfolio, order *entities.Order, limits *RiskLimits) error
}

type Trader interface {
	PlaceOrder(ctx context.Context, order *entities.Order) (*TradeResult, error)
	CancelOrder(ctx context.Context, orderID entities.OrderID) error
	GetOrderStatus(ctx context.Context, orderID entities.OrderID) (*entities.Order, error)
	GetAccountInfo(ctx context.Context) (*AccountInfo, error)
	GetMarketData(ctx context.Context, symbol entities.Symbol) (*entities.MarketData, error)
}

type PriceProvider interface {
	GetRealTimePrice(ctx context.Context, symbol entities.Symbol) (*entities.MarketData, error)
	SubscribeToPrice(ctx context.Context, symbol entities.Symbol, callback func(*entities.MarketData)) error
	UnsubscribeFromPrice(ctx context.Context, symbol entities.Symbol) error
}

type NewsProvider interface {
	GetLatestNews(ctx context.Context, symbols []entities.Symbol) ([]*entities.NewsArticle, error)
	SubscribeToNews(ctx context.Context, callback func(*entities.NewsArticle)) error
}

type Validator interface {
	ValidateOrder(order *entities.Order) error
	ValidateMarketData(data *entities.MarketData) error
	ValidatePortfolio(portfolio *entities.Portfolio) error
	ValidatePosition(position *entities.Position) error
	ValidateNewsArticle(article *entities.NewsArticle) error
	ValidateMacroIndicator(indicator *entities.MacroIndicator) error
	ValidateStruct(s interface{}) error
}

type RiskMetrics struct {
	VaR           float64
	ExpectedLoss  float64
	PositionSize  float64
	LeverageRatio float64
}

type PortfolioRisk struct {
	TotalVaR      float64
	Concentration map[entities.Symbol]float64
	Leverage      float64
	DrawdownRisk  float64
}

type RiskLimits struct {
	MaxPositionSize      float64
	MaxConcentration     float64
	MaxLeverage          float64
	MaxDailyLoss         float64
	MaxVaR               float64
	VaRConfidenceLevel   float64
}

type TradeResult struct {
	OrderID       entities.OrderID
	ExecutedPrice float64
	ExecutedQty   float64
	Commission    float64
	Status        entities.OrderStatus
}

type AccountInfo struct {
	Cash         float64
	MarginUsed   float64
	MarginFree   float64
	Equity       float64
}