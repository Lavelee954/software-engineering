package interfaces

import (
	"context"
	"time"

	"github.com/system-trading/core/internal/entities"
)

type OrderRepository interface {
	Create(ctx context.Context, order *entities.Order) error
	GetByID(ctx context.Context, id entities.OrderID) (*entities.Order, error)
	Update(ctx context.Context, order *entities.Order) error
	List(ctx context.Context, filters OrderFilters) ([]*entities.Order, error)
	Delete(ctx context.Context, id entities.OrderID) error
}

type PortfolioRepository interface {
	Save(ctx context.Context, portfolio *entities.Portfolio) error
	GetByID(ctx context.Context, id string) (*entities.Portfolio, error)
	UpdatePositions(ctx context.Context, portfolio *entities.Portfolio) error
}

type MarketDataRepository interface {
	SaveMarketData(ctx context.Context, data *entities.MarketData) error
	GetLatestMarketData(ctx context.Context, symbol entities.Symbol) (*entities.MarketData, error)
	GetMarketDataHistory(ctx context.Context, symbol entities.Symbol, from, to time.Time) ([]*entities.MarketData, error)
	SaveNewsArticle(ctx context.Context, article *entities.NewsArticle) error
	GetNewsArticles(ctx context.Context, symbols []entities.Symbol, from time.Time) ([]*entities.NewsArticle, error)
	SaveMacroIndicator(ctx context.Context, indicator *entities.MacroIndicator) error
	GetMacroIndicators(ctx context.Context, names []string, from time.Time) ([]*entities.MacroIndicator, error)
}

type AnalysisRepository interface {
	SaveTechnicalIndicators(ctx context.Context, indicators *entities.TechnicalIndicator) error
	GetTechnicalIndicators(ctx context.Context, symbol entities.Symbol, from time.Time) ([]*entities.TechnicalIndicator, error)
	SaveSentimentScore(ctx context.Context, sentiment *entities.SentimentScore) error
	GetSentimentScores(ctx context.Context, symbol entities.Symbol, from time.Time) ([]*entities.SentimentScore, error)
}

type OrderFilters struct {
	Symbol     *entities.Symbol
	Status     *entities.OrderStatus
	Side       *entities.OrderSide
	DateFrom   *time.Time
	DateTo     *time.Time
	Limit      int
	Offset     int
}