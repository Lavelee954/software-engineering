package usecases

import (
	"context"
	"fmt"
	"time"

	"github.com/system-trading/core/internal/entities"
	"github.com/system-trading/core/internal/usecases/interfaces"
)

type OrderService struct {
	orderRepo  interfaces.OrderRepository
	messageBus interfaces.MessageBus
	logger     interfaces.Logger
	metrics    interfaces.MetricsCollector
	validator  interfaces.Validator
}

func NewOrderService(
	orderRepo interfaces.OrderRepository,
	messageBus interfaces.MessageBus,
	logger interfaces.Logger,
	metrics interfaces.MetricsCollector,
	validator interfaces.Validator,
) *OrderService {
	return &OrderService{
		orderRepo:  orderRepo,
		messageBus: messageBus,
		logger:     logger,
		metrics:    metrics,
		validator:  validator,
	}
}

func (s *OrderService) CreateOrder(ctx context.Context, req CreateOrderRequest) (*entities.Order, error) {
	start := time.Now()
	defer func() {
		s.metrics.RecordDuration("order_creation_duration", time.Since(start).Seconds(), map[string]string{
			"symbol": string(req.Symbol),
			"side":   string(req.Side),
		})
	}()

	if err := s.validateCreateOrderRequest(req); err != nil {
		s.metrics.IncrementCounter("order_validation_errors", map[string]string{
			"symbol": string(req.Symbol),
			"error":  "validation_failed",
		})
		s.logger.Warn("Order validation failed",
			interfaces.Field{Key: "symbol", Value: req.Symbol},
			interfaces.Field{Key: "error", Value: err},
		)
		return nil, fmt.Errorf("order validation failed: %w", err)
	}

	order := entities.NewOrder(req.Symbol, req.Side, req.Type, req.Quantity, req.Price)

	if err := s.orderRepo.Create(ctx, order); err != nil {
		s.metrics.IncrementCounter("order_creation_errors", map[string]string{
			"symbol": string(req.Symbol),
			"error":  "repository_failed",
		})
		s.logger.Error("Failed to create order",
			interfaces.Field{Key: "order_id", Value: order.ID},
			interfaces.Field{Key: "error", Value: err},
		)
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	if err := s.publishOrderProposed(ctx, order); err != nil {
		s.logger.Warn("Failed to publish order proposed message",
			interfaces.Field{Key: "order_id", Value: order.ID},
			interfaces.Field{Key: "error", Value: err},
		)
	}

	s.metrics.IncrementCounter("orders_created", map[string]string{
		"symbol": string(req.Symbol),
		"side":   string(req.Side),
		"type":   string(req.Type),
	})

	s.logger.Info("Order created successfully",
		interfaces.Field{Key: "order_id", Value: order.ID},
		interfaces.Field{Key: "symbol", Value: req.Symbol},
		interfaces.Field{Key: "side", Value: req.Side},
		interfaces.Field{Key: "quantity", Value: req.Quantity},
	)

	return order, nil
}

func (s *OrderService) GetOrder(ctx context.Context, orderID entities.OrderID) (*entities.Order, error) {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		s.logger.Error("Failed to get order",
			interfaces.Field{Key: "order_id", Value: orderID},
			interfaces.Field{Key: "error", Value: err},
		)
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	return order, nil
}

func (s *OrderService) UpdateOrderStatus(ctx context.Context, orderID entities.OrderID, status entities.OrderStatus) error {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}

	switch status {
	case entities.OrderStatusApproved:
		order.Approve()
	case entities.OrderStatusRejected:
		order.Reject()
	case entities.OrderStatusCancelled:
		order.Cancel()
	default:
		return fmt.Errorf("invalid status transition to: %s", status)
	}

	if err := s.orderRepo.Update(ctx, order); err != nil {
		s.logger.Error("Failed to update order status",
			interfaces.Field{Key: "order_id", Value: orderID},
			interfaces.Field{Key: "status", Value: status},
			interfaces.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to update order: %w", err)
	}

	if status == entities.OrderStatusApproved {
		if err := s.publishOrderApproved(ctx, order); err != nil {
			s.logger.Warn("Failed to publish order approved message",
				interfaces.Field{Key: "order_id", Value: orderID},
				interfaces.Field{Key: "error", Value: err},
			)
		}
	}

	s.metrics.IncrementCounter("order_status_updates", map[string]string{
		"status": string(status),
	})

	s.logger.Info("Order status updated",
		interfaces.Field{Key: "order_id", Value: orderID},
		interfaces.Field{Key: "status", Value: status},
	)

	return nil
}

func (s *OrderService) ExecuteOrder(ctx context.Context, orderID entities.OrderID, executedPrice, executedQuantity float64) error {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}

	if order.Status != entities.OrderStatusApproved {
		return fmt.Errorf("order must be approved before execution, current status: %s", order.Status)
	}

	order.Execute(executedPrice, executedQuantity)

	if err := s.orderRepo.Update(ctx, order); err != nil {
		s.logger.Error("Failed to update executed order",
			interfaces.Field{Key: "order_id", Value: orderID},
			interfaces.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to update executed order: %w", err)
	}

	if err := s.publishOrderExecuted(ctx, order); err != nil {
		s.logger.Warn("Failed to publish order executed message",
			interfaces.Field{Key: "order_id", Value: orderID},
			interfaces.Field{Key: "error", Value: err},
		)
	}

	s.metrics.IncrementCounter("orders_executed", map[string]string{
		"symbol": string(order.Symbol),
		"side":   string(order.Side),
	})

	s.logger.Info("Order executed successfully",
		interfaces.Field{Key: "order_id", Value: orderID},
		interfaces.Field{Key: "executed_price", Value: executedPrice},
		interfaces.Field{Key: "executed_quantity", Value: executedQuantity},
	)

	return nil
}

func (s *OrderService) ListOrders(ctx context.Context, filters interfaces.OrderFilters) ([]*entities.Order, error) {
	orders, err := s.orderRepo.List(ctx, filters)
	if err != nil {
		s.logger.Error("Failed to list orders",
			interfaces.Field{Key: "error", Value: err},
		)
		return nil, fmt.Errorf("failed to list orders: %w", err)
	}

	return orders, nil
}

func (s *OrderService) CancelOrder(ctx context.Context, orderID entities.OrderID) error {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}

	if order.Status == entities.OrderStatusExecuted {
		return entities.ErrOrderAlreadyExecuted
	}

	if order.Status == entities.OrderStatusCancelled {
		return entities.ErrOrderAlreadyCancelled
	}

	order.Cancel()

	if err := s.orderRepo.Update(ctx, order); err != nil {
		s.logger.Error("Failed to cancel order",
			interfaces.Field{Key: "order_id", Value: orderID},
			interfaces.Field{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to cancel order: %w", err)
	}

	s.metrics.IncrementCounter("orders_cancelled", map[string]string{
		"symbol": string(order.Symbol),
	})

	s.logger.Info("Order cancelled",
		interfaces.Field{Key: "order_id", Value: orderID},
	)

	return nil
}

func (s *OrderService) validateCreateOrderRequest(req CreateOrderRequest) error {
	if req.Symbol == "" {
		return fmt.Errorf("symbol is required")
	}

	if req.Side != entities.OrderSideBuy && req.Side != entities.OrderSideSell {
		return fmt.Errorf("invalid order side: %s", req.Side)
	}

	if req.Type != entities.OrderTypeMarket && 
	   req.Type != entities.OrderTypeLimit && 
	   req.Type != entities.OrderTypeStop {
		return fmt.Errorf("invalid order type: %s", req.Type)
	}

	if req.Quantity <= 0 {
		return fmt.Errorf("quantity must be positive")
	}

	if (req.Type == entities.OrderTypeLimit || req.Type == entities.OrderTypeStop) && req.Price == nil {
		return fmt.Errorf("price is required for %s orders", req.Type)
	}

	if req.Price != nil && *req.Price <= 0 {
		return fmt.Errorf("price must be positive")
	}

	return nil
}

func (s *OrderService) publishOrderProposed(ctx context.Context, order *entities.Order) error {
	return s.messageBus.Publish(ctx, "order.proposed", order)
}

func (s *OrderService) publishOrderApproved(ctx context.Context, order *entities.Order) error {
	return s.messageBus.Publish(ctx, "order.approved", order)
}

func (s *OrderService) publishOrderExecuted(ctx context.Context, order *entities.Order) error {
	return s.messageBus.Publish(ctx, "order.executed", order)
}

type CreateOrderRequest struct {
	Symbol   entities.Symbol    `json:"symbol" validate:"required,symbol"`
	Side     entities.OrderSide `json:"side" validate:"required"`
	Type     entities.OrderType `json:"type" validate:"required"`
	Quantity float64           `json:"quantity" validate:"required,min=0.000001"`
	Price    *float64          `json:"price,omitempty" validate:"omitempty,min=0.000001"`
}