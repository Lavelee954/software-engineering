package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/system-trading/core/internal/entities"
	"github.com/system-trading/core/internal/usecases/interfaces"
)

type PortfolioService struct {
	portfolioRepo interfaces.PortfolioRepository
	messageBus    interfaces.MessageBus
	logger        interfaces.Logger
	metrics       interfaces.MetricsCollector
}

func NewPortfolioService(
	portfolioRepo interfaces.PortfolioRepository,
	messageBus interfaces.MessageBus,
	logger interfaces.Logger,
	metrics interfaces.MetricsCollector,
) *PortfolioService {
	return &PortfolioService{
		portfolioRepo: portfolioRepo,
		messageBus:    messageBus,
		logger:        logger,
		metrics:       metrics,
	}
}

func (s *PortfolioService) CreatePortfolio(ctx context.Context, initialCash float64) (*entities.Portfolio, error) {
	if initialCash < 0 {
		return nil, fmt.Errorf("initial cash cannot be negative: %f", initialCash)
	}

	portfolio := entities.NewPortfolio(initialCash)

	if err := s.portfolioRepo.Save(ctx, portfolio); err != nil {
		s.logger.Error("Failed to create portfolio",
			interfaces.Field{Key: "portfolio_id", Value: portfolio.ID},
			interfaces.Field{Key: "error", Value: err},
		)
		return nil, fmt.Errorf("failed to create portfolio: %w", err)
	}

	s.metrics.SetGauge("portfolio_value", portfolio.TotalValue, map[string]string{
		"portfolio_id": portfolio.ID,
	})

	s.logger.Info("Portfolio created",
		interfaces.Field{Key: "portfolio_id", Value: portfolio.ID},
		interfaces.Field{Key: "initial_cash", Value: initialCash},
	)

	return portfolio, nil
}

func (s *PortfolioService) GetPortfolio(ctx context.Context, portfolioID string) (*entities.Portfolio, error) {
	portfolio, err := s.portfolioRepo.GetByID(ctx, portfolioID)
	if err != nil {
		s.logger.Error("Failed to get portfolio",
			interfaces.Field{Key: "portfolio_id", Value: portfolioID},
			interfaces.Field{Key: "error", Value: err},
		)
		return nil, fmt.Errorf("failed to get portfolio: %w", err)
	}

	return portfolio, nil
}

func (s *PortfolioService) ProcessOrderExecution(ctx context.Context, order *entities.Order) error {
	if order.Status != entities.OrderStatusExecuted {
		return fmt.Errorf("order must be executed status, got: %s", order.Status)
	}

	if order.ExecutedPrice == nil || order.ExecutedQuantity == nil {
		return fmt.Errorf("executed price and quantity are required")
	}

	portfolio, err := s.getDefaultPortfolio(ctx)
	if err != nil {
		return fmt.Errorf("failed to get portfolio: %w", err)
	}

	switch order.Side {
	case entities.OrderSideBuy:
		err = s.processBuyOrder(portfolio, order)
	case entities.OrderSideSell:
		err = s.processSellOrder(portfolio, order)
	default:
		return fmt.Errorf("invalid order side: %s", order.Side)
	}

	if err != nil {
		s.logger.Error("Failed to process order execution",
			interfaces.Field{Key: "order_id", Value: order.ID},
			interfaces.Field{Key: "portfolio_id", Value: portfolio.ID},
			interfaces.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to process order execution: %w", err)
	}

	if err := s.portfolioRepo.UpdatePositions(ctx, portfolio); err != nil {
		s.logger.Error("Failed to update portfolio positions",
			interfaces.Field{Key: "portfolio_id", Value: portfolio.ID},
			interfaces.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to update portfolio: %w", err)
	}

	if err := s.publishPortfolioUpdate(ctx, portfolio); err != nil {
		s.logger.Warn("Failed to publish portfolio update",
			interfaces.Field{Key: "portfolio_id", Value: portfolio.ID},
			interfaces.Field{Key: "error", Value: err},
		)
	}

	s.updatePortfolioMetrics(portfolio)

	s.logger.Info("Order execution processed",
		interfaces.Field{Key: "order_id", Value: order.ID},
		interfaces.Field{Key: "portfolio_id", Value: portfolio.ID},
		interfaces.Field{Key: "symbol", Value: order.Symbol},
		interfaces.Field{Key: "side", Value: order.Side},
		interfaces.Field{Key: "executed_quantity", Value: *order.ExecutedQuantity},
		interfaces.Field{Key: "executed_price", Value: *order.ExecutedPrice},
	)

	return nil
}

func (s *PortfolioService) UpdatePositionPrices(ctx context.Context, portfolioID string, marketData *entities.MarketData) error {
	portfolio, err := s.portfolioRepo.GetByID(ctx, portfolioID)
	if err != nil {
		return fmt.Errorf("failed to get portfolio: %w", err)
	}

	portfolio.UpdatePositionPrice(marketData.Symbol, marketData.Price)

	if err := s.portfolioRepo.UpdatePositions(ctx, portfolio); err != nil {
		s.logger.Error("Failed to update position prices",
			interfaces.Field{Key: "portfolio_id", Value: portfolioID},
			interfaces.Field{Key: "symbol", Value: marketData.Symbol},
			interfaces.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to update position prices: %w", err)
	}

	s.updatePortfolioMetrics(portfolio)

	return nil
}

func (s *PortfolioService) GetPortfolioPositions(ctx context.Context, portfolioID string) (map[entities.Symbol]*entities.Position, error) {
	portfolio, err := s.portfolioRepo.GetByID(ctx, portfolioID)
	if err != nil {
		return nil, fmt.Errorf("failed to get portfolio: %w", err)
	}

	return portfolio.Positions, nil
}

func (s *PortfolioService) GetPortfolioPerformance(ctx context.Context, portfolioID string) (*PortfolioPerformance, error) {
	portfolio, err := s.portfolioRepo.GetByID(ctx, portfolioID)
	if err != nil {
		return nil, fmt.Errorf("failed to get portfolio: %w", err)
	}

	totalUnrealizedPnL := 0.0
	totalRealizedPnL := 0.0
	positionCount := len(portfolio.Positions)

	for _, position := range portfolio.Positions {
		totalUnrealizedPnL += position.UnrealizedPnL
		totalRealizedPnL += position.RealizedPnL
	}

	performance := &PortfolioPerformance{
		PortfolioID:       portfolioID,
		TotalValue:        portfolio.TotalValue,
		Cash:              portfolio.Cash,
		TotalPnL:          portfolio.TotalPnL,
		UnrealizedPnL:     totalUnrealizedPnL,
		RealizedPnL:       totalRealizedPnL,
		PositionCount:     positionCount,
		LastUpdated:       portfolio.LastUpdated,
	}

	return performance, nil
}

func (s *PortfolioService) processBuyOrder(portfolio *entities.Portfolio, order *entities.Order) error {
	totalCost := (*order.ExecutedQuantity) * (*order.ExecutedPrice)

	if portfolio.Cash < totalCost {
		return entities.ErrInsufficientCash
	}

	portfolio.AddPosition(order.Symbol, *order.ExecutedQuantity, *order.ExecutedPrice)

	s.metrics.IncrementCounter("buy_orders_processed", map[string]string{
		"symbol": string(order.Symbol),
	})

	return nil
}

func (s *PortfolioService) processSellOrder(portfolio *entities.Portfolio, order *entities.Order) error {
	position, exists := portfolio.GetPosition(order.Symbol)
	if !exists {
		return entities.ErrPositionNotFound
	}

	if position.Quantity < *order.ExecutedQuantity {
		return entities.ErrInsufficientQuantity
	}

	if err := portfolio.RemovePosition(order.Symbol, *order.ExecutedQuantity, *order.ExecutedPrice); err != nil {
		return err
	}

	s.metrics.IncrementCounter("sell_orders_processed", map[string]string{
		"symbol": string(order.Symbol),
	})

	return nil
}

func (s *PortfolioService) getDefaultPortfolio(ctx context.Context) (*entities.Portfolio, error) {
	return s.portfolioRepo.GetByID(ctx, "default")
}

func (s *PortfolioService) publishPortfolioUpdate(ctx context.Context, portfolio *entities.Portfolio) error {
	update := PortfolioUpdateMessage{
		PortfolioID: portfolio.ID,
		TotalValue:  portfolio.TotalValue,
		Cash:        portfolio.Cash,
		TotalPnL:    portfolio.TotalPnL,
		DayPnL:      portfolio.DayPnL,
		Positions:   make(map[entities.Symbol]float64),
		Timestamp:   time.Now(),
	}

	for symbol, position := range portfolio.Positions {
		update.Positions[symbol] = position.Quantity
	}

	return s.messageBus.Publish(ctx, "portfolio.update", update)
}

func (s *PortfolioService) updatePortfolioMetrics(portfolio *entities.Portfolio) {
	s.metrics.SetGauge("portfolio_value", portfolio.TotalValue, map[string]string{
		"portfolio_id": portfolio.ID,
	})

	s.metrics.SetGauge("portfolio_cash", portfolio.Cash, map[string]string{
		"portfolio_id": portfolio.ID,
	})

	s.metrics.SetGauge("portfolio_total_pnl", portfolio.TotalPnL, map[string]string{
		"portfolio_id": portfolio.ID,
	})

	s.metrics.SetGauge("portfolio_day_pnl", portfolio.DayPnL, map[string]string{
		"portfolio_id": portfolio.ID,
	})

	s.metrics.SetGauge("position_count", float64(len(portfolio.Positions)), map[string]string{
		"portfolio_id": portfolio.ID,
	})

	for symbol, position := range portfolio.Positions {
		s.metrics.SetGauge("position_value", position.MarketValue, map[string]string{
			"portfolio_id": portfolio.ID,
			"symbol":       string(symbol),
		})

		s.metrics.SetGauge("position_unrealized_pnl", position.UnrealizedPnL, map[string]string{
			"portfolio_id": portfolio.ID,
			"symbol":       string(symbol),
		})
	}
}

type PortfolioPerformance struct {
	PortfolioID   string    `json:"portfolio_id"`
	TotalValue    float64   `json:"total_value"`
	Cash          float64   `json:"cash"`
	TotalPnL      float64   `json:"total_pnl"`
	UnrealizedPnL float64   `json:"unrealized_pnl"`
	RealizedPnL   float64   `json:"realized_pnl"`
	PositionCount int       `json:"position_count"`
	LastUpdated   time.Time `json:"last_updated"`
}

type PortfolioUpdateMessage struct {
	PortfolioID string                      `json:"portfolio_id"`
	TotalValue  float64                     `json:"total_value"`
	Cash        float64                     `json:"cash"`
	TotalPnL    float64                     `json:"total_pnl"`
	DayPnL      float64                     `json:"day_pnl"`
	Positions   map[entities.Symbol]float64 `json:"positions"`
	Timestamp   time.Time                   `json:"timestamp"`
}