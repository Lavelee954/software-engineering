package usecases

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/system-trading/core/internal/entities"
	"github.com/system-trading/core/internal/usecases/interfaces"
)

type RiskService struct {
	portfolioService *PortfolioService
	messageBus       interfaces.MessageBus
	logger           interfaces.Logger
	metrics          interfaces.MetricsCollector
	riskLimits       *interfaces.RiskLimits
}

func NewRiskService(
	portfolioService *PortfolioService,
	messageBus interfaces.MessageBus,
	logger interfaces.Logger,
	metrics interfaces.MetricsCollector,
	riskLimits *interfaces.RiskLimits,
) *RiskService {
	return &RiskService{
		portfolioService: portfolioService,
		messageBus:       messageBus,
		logger:           logger,
		metrics:          metrics,
		riskLimits:       riskLimits,
	}
}

func (s *RiskService) ValidateOrder(ctx context.Context, order *entities.Order) error {
	start := time.Now()
	defer func() {
		s.metrics.RecordDuration("risk_validation_duration", time.Since(start).Seconds(), map[string]string{
			"symbol": string(order.Symbol),
		})
	}()

	portfolio, err := s.portfolioService.GetPortfolio(ctx, "default")
	if err != nil {
		s.logger.Error("Failed to get portfolio for risk validation",
			interfaces.Field{Key: "order_id", Value: order.ID},
			interfaces.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to get portfolio: %w", err)
	}

	if err := s.validateCashBalance(portfolio, order); err != nil {
		s.publishRiskAlert(ctx, "INSUFFICIENT_CASH", "HIGH", order.Symbol, err.Error())
		return err
	}

	if err := s.validatePositionSize(portfolio, order); err != nil {
		s.publishRiskAlert(ctx, "POSITION_SIZE_LIMIT", "HIGH", order.Symbol, err.Error())
		return err
	}

	if err := s.validateConcentration(portfolio, order); err != nil {
		s.publishRiskAlert(ctx, "CONCENTRATION_LIMIT", "MEDIUM", order.Symbol, err.Error())
		return err
	}

	if err := s.validateVaRLimit(portfolio, order); err != nil {
		s.publishRiskAlert(ctx, "VAR_LIMIT", "HIGH", order.Symbol, err.Error())
		return err
	}

	if err := s.validateDailyLossLimit(portfolio); err != nil {
		s.publishRiskAlert(ctx, "DAILY_LOSS_LIMIT", "CRITICAL", order.Symbol, err.Error())
		return err
	}

	s.metrics.IncrementCounter("risk_validations_passed", map[string]string{
		"symbol": string(order.Symbol),
		"side":   string(order.Side),
	})

	s.logger.Info("Order passed risk validation",
		interfaces.Field{Key: "order_id", Value: order.ID},
		interfaces.Field{Key: "symbol", Value: order.Symbol},
	)

	return nil
}

func (s *RiskService) CalculatePortfolioRisk(ctx context.Context, portfolioID string) (*interfaces.PortfolioRisk, error) {
	portfolio, err := s.portfolioService.GetPortfolio(ctx, portfolioID)
	if err != nil {
		return nil, fmt.Errorf("failed to get portfolio: %w", err)
	}

	var95 := s.calculateVaR(portfolio, 0.95)
	var99 := s.calculateVaR(portfolio, 0.99)
	leverage := s.calculateLeverage(portfolio)
	concentration := s.calculateConcentration(portfolio)
	drawdownRisk := s.calculateDrawdownRisk(portfolio)

	portfolioRisk := &interfaces.PortfolioRisk{
		TotalVaR:      var95,
		Concentration: concentration,
		Leverage:      leverage,
		DrawdownRisk:  drawdownRisk,
	}

	s.updateRiskMetrics(portfolioID, var95, var99, leverage)

	return portfolioRisk, nil
}

func (s *RiskService) CalculatePositionRisk(ctx context.Context, portfolioID string, symbol entities.Symbol) (*interfaces.RiskMetrics, error) {
	portfolio, err := s.portfolioService.GetPortfolio(ctx, portfolioID)
	if err != nil {
		return nil, fmt.Errorf("failed to get portfolio: %w", err)
	}

	position, exists := portfolio.GetPosition(symbol)
	if !exists {
		return nil, entities.ErrPositionNotFound
	}

	var95 := s.calculatePositionVaR(position, 0.95)
	expectedLoss := s.calculateExpectedLoss(position)
	positionSize := position.MarketValue / portfolio.TotalValue
	leverageRatio := s.calculatePositionLeverage(position, portfolio)

	riskMetrics := &interfaces.RiskMetrics{
		VaR:           var95,
		ExpectedLoss:  expectedLoss,
		PositionSize:  positionSize,
		LeverageRatio: leverageRatio,
	}

	return riskMetrics, nil
}

func (s *RiskService) MonitorRiskLimits(ctx context.Context, portfolioID string) error {
	portfolioRisk, err := s.CalculatePortfolioRisk(ctx, portfolioID)
	if err != nil {
		return fmt.Errorf("failed to calculate portfolio risk: %w", err)
	}

	if portfolioRisk.TotalVaR > s.riskLimits.MaxVaR {
		s.publishRiskAlert(ctx, "VAR_EXCEEDED", "CRITICAL", "", 
			fmt.Sprintf("Portfolio VaR (%.4f) exceeds limit (%.4f)", portfolioRisk.TotalVaR, s.riskLimits.MaxVaR))
	}

	if portfolioRisk.Leverage > s.riskLimits.MaxLeverage {
		s.publishRiskAlert(ctx, "LEVERAGE_EXCEEDED", "HIGH", "", 
			fmt.Sprintf("Portfolio leverage (%.2f) exceeds limit (%.2f)", portfolioRisk.Leverage, s.riskLimits.MaxLeverage))
	}

	for symbol, concentration := range portfolioRisk.Concentration {
		if concentration > s.riskLimits.MaxConcentration {
			s.publishRiskAlert(ctx, "CONCENTRATION_EXCEEDED", "MEDIUM", symbol, 
				fmt.Sprintf("Position concentration (%.2f%%) exceeds limit (%.2f%%)", 
					concentration*100, s.riskLimits.MaxConcentration*100))
		}
	}

	return nil
}

func (s *RiskService) validateCashBalance(portfolio *entities.Portfolio, order *entities.Order) error {
	if order.Side != entities.OrderSideBuy {
		return nil
	}

	var requiredCash float64
	if order.Type == entities.OrderTypeMarket {
		requiredCash = order.Quantity * s.estimateMarketPrice(order.Symbol)
	} else if order.Price != nil {
		requiredCash = order.Quantity * (*order.Price)
	} else {
		return fmt.Errorf("price is required for limit orders")
	}

	cashBuffer := requiredCash * 0.1
	totalRequired := requiredCash + cashBuffer

	if portfolio.Cash < totalRequired {
		s.metrics.IncrementCounter("risk_violations", map[string]string{
			"type":   "insufficient_cash",
			"symbol": string(order.Symbol),
		})
		return fmt.Errorf("insufficient cash: required %.2f, available %.2f", totalRequired, portfolio.Cash)
	}

	return nil
}

func (s *RiskService) validatePositionSize(portfolio *entities.Portfolio, order *entities.Order) error {
	if order.Side != entities.OrderSideBuy {
		return nil
	}

	var orderValue float64
	if order.Type == entities.OrderTypeMarket {
		orderValue = order.Quantity * s.estimateMarketPrice(order.Symbol)
	} else if order.Price != nil {
		orderValue = order.Quantity * (*order.Price)
	}

	currentPosition, exists := portfolio.GetPosition(order.Symbol)
	currentValue := 0.0
	if exists {
		currentValue = currentPosition.MarketValue
	}

	newPositionValue := currentValue + orderValue
	positionSizeRatio := newPositionValue / portfolio.TotalValue

	if positionSizeRatio > s.riskLimits.MaxPositionSize {
		s.metrics.IncrementCounter("risk_violations", map[string]string{
			"type":   "position_size",
			"symbol": string(order.Symbol),
		})
		return fmt.Errorf("position size limit exceeded: %.2f%% > %.2f%%", 
			positionSizeRatio*100, s.riskLimits.MaxPositionSize*100)
	}

	return nil
}

func (s *RiskService) validateConcentration(portfolio *entities.Portfolio, order *entities.Order) error {
	concentration := s.calculateConcentration(portfolio)
	
	for symbol, ratio := range concentration {
		if ratio > s.riskLimits.MaxConcentration {
			s.metrics.IncrementCounter("risk_violations", map[string]string{
				"type":   "concentration",
				"symbol": string(symbol),
			})
			return fmt.Errorf("concentration limit exceeded for %s: %.2f%% > %.2f%%", 
				symbol, ratio*100, s.riskLimits.MaxConcentration*100)
		}
	}

	return nil
}

func (s *RiskService) validateVaRLimit(portfolio *entities.Portfolio, order *entities.Order) error {
	currentVaR := s.calculateVaR(portfolio, s.riskLimits.VaRConfidenceLevel)
	
	if currentVaR > s.riskLimits.MaxVaR {
		s.metrics.IncrementCounter("risk_violations", map[string]string{
			"type":   "var_limit",
			"symbol": string(order.Symbol),
		})
		return fmt.Errorf("VaR limit exceeded: %.4f > %.4f", currentVaR, s.riskLimits.MaxVaR)
	}

	return nil
}

func (s *RiskService) validateDailyLossLimit(portfolio *entities.Portfolio) error {
	dailyLossRatio := math.Abs(portfolio.DayPnL) / portfolio.TotalValue
	
	if dailyLossRatio > s.riskLimits.MaxDailyLoss {
		s.metrics.IncrementCounter("risk_violations", map[string]string{
			"type": "daily_loss",
		})
		return fmt.Errorf("daily loss limit exceeded: %.2f%% > %.2f%%", 
			dailyLossRatio*100, s.riskLimits.MaxDailyLoss*100)
	}

	return nil
}

func (s *RiskService) calculateVaR(portfolio *entities.Portfolio, confidenceLevel float64) float64 {
	if len(portfolio.Positions) == 0 {
		return 0.0
	}

	totalRisk := 0.0
	for _, position := range portfolio.Positions {
		positionRisk := s.calculatePositionVaR(position, confidenceLevel)
		totalRisk += positionRisk * positionRisk
	}

	return math.Sqrt(totalRisk)
}

func (s *RiskService) calculatePositionVaR(position *entities.Position, confidenceLevel float64) float64 {
	volatility := s.estimateVolatility(position.Symbol)
	zScore := s.getZScore(confidenceLevel)
	
	return position.MarketValue * volatility * zScore
}

func (s *RiskService) calculateLeverage(portfolio *entities.Portfolio) float64 {
	if portfolio.TotalValue == 0 {
		return 0.0
	}

	totalPositionValue := 0.0
	for _, position := range portfolio.Positions {
		totalPositionValue += math.Abs(position.MarketValue)
	}

	return totalPositionValue / portfolio.TotalValue
}

func (s *RiskService) calculateConcentration(portfolio *entities.Portfolio) map[entities.Symbol]float64 {
	concentration := make(map[entities.Symbol]float64)
	
	if portfolio.TotalValue == 0 {
		return concentration
	}

	for symbol, position := range portfolio.Positions {
		concentration[symbol] = math.Abs(position.MarketValue) / portfolio.TotalValue
	}

	return concentration
}

func (s *RiskService) calculateDrawdownRisk(portfolio *entities.Portfolio) float64 {
	maxDrawdown := 0.0
	for _, position := range portfolio.Positions {
		if position.UnrealizedPnL < 0 {
			drawdown := math.Abs(position.UnrealizedPnL) / position.MarketValue
			if drawdown > maxDrawdown {
				maxDrawdown = drawdown
			}
		}
	}
	return maxDrawdown
}

func (s *RiskService) calculateExpectedLoss(position *entities.Position) float64 {
	volatility := s.estimateVolatility(position.Symbol)
	return position.MarketValue * volatility * 0.5
}

func (s *RiskService) calculatePositionLeverage(position *entities.Position, portfolio *entities.Portfolio) float64 {
	if portfolio.TotalValue == 0 {
		return 0.0
	}
	return math.Abs(position.MarketValue) / portfolio.TotalValue
}

func (s *RiskService) estimateMarketPrice(symbol entities.Symbol) float64 {
	return 100.0
}

func (s *RiskService) estimateVolatility(symbol entities.Symbol) float64 {
	return 0.02
}

func (s *RiskService) getZScore(confidenceLevel float64) float64 {
	zScores := map[float64]float64{
		0.90: 1.28,
		0.95: 1.645,
		0.99: 2.33,
	}
	
	if z, exists := zScores[confidenceLevel]; exists {
		return z
	}
	return 1.645
}

func (s *RiskService) publishRiskAlert(ctx context.Context, alertType, severity string, symbol entities.Symbol, message string) {
	alert := RiskAlertMessage{
		AlertType:   alertType,
		Severity:    severity,
		Symbol:      symbol,
		Message:     message,
		Timestamp:   time.Now(),
	}

	if err := s.messageBus.Publish(ctx, "risk.alert", alert); err != nil {
		s.logger.Error("Failed to publish risk alert",
			interfaces.Field{Key: "alert_type", Value: alertType},
			interfaces.Field{Key: "error", Value: err},
		)
	}

	s.metrics.IncrementCounter("risk_alerts", map[string]string{
		"type":     alertType,
		"severity": severity,
	})
}

func (s *RiskService) updateRiskMetrics(portfolioID string, var95, var99, leverage float64) {
	s.metrics.SetGauge("portfolio_var_95", var95, map[string]string{
		"portfolio_id": portfolioID,
	})

	s.metrics.SetGauge("portfolio_var_99", var99, map[string]string{
		"portfolio_id": portfolioID,
	})

	s.metrics.SetGauge("portfolio_leverage", leverage, map[string]string{
		"portfolio_id": portfolioID,
	})
}

type RiskAlertMessage struct {
	AlertType string          `json:"alert_type"`
	Severity  string          `json:"severity"`
	Symbol    entities.Symbol `json:"symbol,omitempty"`
	Message   string          `json:"message"`
	Timestamp time.Time       `json:"timestamp"`
}