package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/system-trading/core/internal/entities"
	"github.com/system-trading/core/internal/interfaces"
	ifs "github.com/system-trading/core/internal/usecases/interfaces"
)

// ExecutionAgent handles order execution through brokerage APIs
type ExecutionAgent struct {
	messageBus      ifs.MessageBus
	trader          interfaces.Trader
	logger          ifs.Logger
	metrics         ifs.MetricsCollector
	orderTracker    map[string]*ExecutionContext
	mu              sync.RWMutex
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	retryConfig     RetryConfig
}

// ExecutionContext tracks the state of an order being executed
type ExecutionContext struct {
	Order           *entities.Order
	BrokerOrderID   string
	SubmittedAt     time.Time
	LastStatusCheck time.Time
	RetryCount      int
	Status          entities.OrderStatus
}

// RetryConfig defines retry behavior for failed operations
type RetryConfig struct {
	MaxRetries      int
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	BackoffFactor   float64
	StatusCheckInterval time.Duration
}

// ExecutedOrderMessage represents a message published when an order is executed
type ExecutedOrderMessage struct {
	OrderID         string    `json:"order_id"`
	BrokerOrderID   string    `json:"broker_order_id"`
	Symbol          string    `json:"symbol"`
	Side            string    `json:"side"`
	Quantity        float64   `json:"quantity"`
	ExecutedPrice   float64   `json:"executed_price"`
	ExecutedQty     float64   `json:"executed_quantity"`
	Fees            float64   `json:"fees"`
	ExecutedAt      time.Time `json:"executed_at"`
	BrokerName      string    `json:"broker_name"`
}

// NewExecutionAgent creates a new execution agent
func NewExecutionAgent(
	messageBus ifs.MessageBus,
	trader interfaces.Trader,
	logger ifs.Logger,
	metrics ifs.MetricsCollector,
) *ExecutionAgent {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &ExecutionAgent{
		messageBus:   messageBus,
		trader:       trader,
		logger:       logger,
		metrics:      metrics,
		orderTracker: make(map[string]*ExecutionContext),
		ctx:          ctx,
		cancel:       cancel,
		retryConfig: RetryConfig{
			MaxRetries:          3,
			InitialDelay:        1 * time.Second,
			MaxDelay:           30 * time.Second,
			BackoffFactor:      2.0,
			StatusCheckInterval: 5 * time.Second,
		},
	}
}

// Start begins the execution agent's operation
func (ea *ExecutionAgent) Start(ctx context.Context) error {
	ea.logger.Info("Starting execution agent",
		ifs.Field{Key: "broker", Value: ea.trader.GetBrokerName()},
	)
	
	// Connect to broker
	if err := ea.trader.Connect(ctx); err != nil {
		ea.metrics.IncrementCounter("execution_agent_errors", map[string]string{
			"type": "broker_connection_failed",
		})
		return fmt.Errorf("failed to connect to broker: %w", err)
	}
	
	// Subscribe to approved orders
	if err := ea.messageBus.Subscribe(ctx, "order.approved", ea.handleApprovedOrder); err != nil {
		return fmt.Errorf("failed to subscribe to order.approved: %w", err)
	}
	
	// Start status monitoring goroutine
	ea.wg.Add(1)
	go ea.monitorOrderStatus()
	
	ea.metrics.IncrementCounter("execution_agent_started", map[string]string{
		"broker": ea.trader.GetBrokerName(),
	})
	
	ea.logger.Info("Execution agent started successfully")
	return nil
}

// Stop gracefully shuts down the execution agent
func (ea *ExecutionAgent) Stop(ctx context.Context) error {
	ea.logger.Info("Stopping execution agent")
	
	// Cancel context to stop all operations
	ea.cancel()
	
	// Disconnect from broker
	if ea.trader.IsConnected() {
		if err := ea.trader.Disconnect(ctx); err != nil {
			ea.logger.Error("Failed to disconnect from broker",
				ifs.Field{Key: "error", Value: err.Error()},
			)
		}
	}
	
	// Wait for goroutines to finish
	done := make(chan struct{})
	go func() {
		ea.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		ea.logger.Info("Execution agent stopped successfully")
		return nil
	case <-time.After(30 * time.Second):
		ea.logger.Warn("Execution agent shutdown timeout")
		return fmt.Errorf("shutdown timeout")
	}
}

// handleApprovedOrder processes approved orders for execution
func (ea *ExecutionAgent) handleApprovedOrder(ctx context.Context, data []byte) error {
	var order entities.Order
	if err := json.Unmarshal(data, &order); err != nil {
		ea.metrics.IncrementCounter("execution_agent_errors", map[string]string{
			"type": "message_decode_failed",
		})
		return fmt.Errorf("failed to unmarshal order: %w", err)
	}
	
	ea.logger.Info("Received approved order for execution",
		ifs.Field{Key: "order_id", Value: string(order.ID)},
		ifs.Field{Key: "symbol", Value: string(order.Symbol)},
		ifs.Field{Key: "side", Value: string(order.Side)},
		ifs.Field{Key: "quantity", Value: order.Quantity},
	)
	
	// Execute the order
	if err := ea.executeOrder(ctx, &order); err != nil {
		ea.logger.Error("Failed to execute order",
			ifs.Field{Key: "order_id", Value: string(order.ID)},
			ifs.Field{Key: "error", Value: err.Error()},
		)
		
		ea.metrics.IncrementCounter("execution_agent_errors", map[string]string{
			"type": "order_execution_failed",
		})
		
		// Publish order failure event
		ea.publishOrderEvent(ctx, "order.failed", &order, nil, err)
		return err
	}
	
	return nil
}

// executeOrder executes a single order
func (ea *ExecutionAgent) executeOrder(ctx context.Context, order *entities.Order) error {
	startTime := time.Now()
	
	defer func() {
		duration := time.Since(startTime)
		ea.metrics.RecordDuration("execution_agent_order_duration", float64(duration.Seconds()), map[string]string{
			"symbol": string(order.Symbol),
			"side":   string(order.Side),
		})
	}()
	
	// Validate order before execution
	if err := ea.validateOrder(order); err != nil {
		return fmt.Errorf("order validation failed: %w", err)
	}
	
	// Attempt to place order with retries
	var result *interfaces.OrderResult
	var err error
	
	for attempt := 0; attempt <= ea.retryConfig.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := ea.calculateRetryDelay(attempt)
			ea.logger.Warn("Retrying order execution",
				ifs.Field{Key: "order_id", Value: string(order.ID)},
				ifs.Field{Key: "attempt", Value: attempt + 1},
				ifs.Field{Key: "delay", Value: delay},
			)
			
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		
		result, err = ea.trader.PlaceOrder(ctx, order)
		if err == nil {
			break
		}
		
		ea.logger.Warn("Order execution attempt failed",
			ifs.Field{Key: "order_id", Value: string(order.ID)},
			ifs.Field{Key: "attempt", Value: attempt + 1},
			ifs.Field{Key: "error", Value: err.Error()},
		)
		
		// Don't retry certain types of errors
		if !ea.isRetryableError(err) {
			break
		}
	}
	
	if err != nil {
		return fmt.Errorf("failed to place order after %d attempts: %w", 
			ea.retryConfig.MaxRetries+1, err)
	}
	
	// Track the order for status monitoring
	ea.trackOrder(order, result.BrokerOrderID)
	
	ea.logger.Info("Order submitted to broker",
		ifs.Field{Key: "order_id", Value: string(order.ID)},
		ifs.Field{Key: "broker_order_id", Value: result.BrokerOrderID},
		ifs.Field{Key: "status", Value: string(result.Status)},
	)
	
	ea.metrics.IncrementCounter("execution_agent_orders_submitted", map[string]string{
		"symbol": string(order.Symbol),
		"side":   string(order.Side),
		"broker": ea.trader.GetBrokerName(),
	})
	
	// For market orders that are immediately executed, publish execution event
	if result.Status == entities.OrderStatusExecuted && result.ExecutedPrice != nil {
		ea.publishExecutedOrder(ctx, order, result)
	}
	
	return nil
}

// validateOrder validates an order before execution
func (ea *ExecutionAgent) validateOrder(order *entities.Order) error {
	if order.ID == "" {
		return fmt.Errorf("order ID is required")
	}
	
	if order.Symbol == "" {
		return fmt.Errorf("order symbol is required")
	}
	
	if order.Quantity <= 0 {
		return fmt.Errorf("order quantity must be positive")
	}
	
	if order.Side != entities.OrderSideBuy && order.Side != entities.OrderSideSell {
		return fmt.Errorf("invalid order side: %s", order.Side)
	}
	
	if order.Type != entities.OrderTypeMarket && order.Type != entities.OrderTypeLimit {
		return fmt.Errorf("unsupported order type: %s", order.Type)
	}
	
	if order.Type == entities.OrderTypeLimit && (order.Price == nil || *order.Price <= 0) {
		return fmt.Errorf("limit orders must have a positive price")
	}
	
	return nil
}

// trackOrder adds an order to the tracking system
func (ea *ExecutionAgent) trackOrder(order *entities.Order, brokerOrderID string) {
	ea.mu.Lock()
	defer ea.mu.Unlock()
	
	ea.orderTracker[brokerOrderID] = &ExecutionContext{
		Order:           order,
		BrokerOrderID:   brokerOrderID,
		SubmittedAt:     time.Now(),
		LastStatusCheck: time.Now(),
		RetryCount:      0,
		Status:          entities.OrderStatusPending,
	}
}

// monitorOrderStatus monitors pending orders and publishes execution events
func (ea *ExecutionAgent) monitorOrderStatus() {
	defer ea.wg.Done()
	
	ticker := time.NewTicker(ea.retryConfig.StatusCheckInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ea.ctx.Done():
			return
		case <-ticker.C:
			ea.checkPendingOrders()
		}
	}
}

// checkPendingOrders checks the status of all pending orders
func (ea *ExecutionAgent) checkPendingOrders() {
	ea.mu.RLock()
	orderIDs := make([]string, 0, len(ea.orderTracker))
	for brokerOrderID := range ea.orderTracker {
		orderIDs = append(orderIDs, brokerOrderID)
	}
	ea.mu.RUnlock()
	
	for _, brokerOrderID := range orderIDs {
		if err := ea.checkOrderStatus(brokerOrderID); err != nil {
			ea.logger.Error("Failed to check order status",
				ifs.Field{Key: "broker_order_id", Value: brokerOrderID},
				ifs.Field{Key: "error", Value: err.Error()},
			)
		}
	}
}

// checkOrderStatus checks the status of a specific order
func (ea *ExecutionAgent) checkOrderStatus(brokerOrderID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	status, err := ea.trader.GetOrderStatus(ctx, brokerOrderID)
	if err != nil {
		return fmt.Errorf("failed to get order status: %w", err)
	}
	
	ea.mu.Lock()
	execCtx, exists := ea.orderTracker[brokerOrderID]
	if !exists {
		ea.mu.Unlock()
		return nil // Order no longer tracked
	}
	
	execCtx.LastStatusCheck = time.Now()
	previousStatus := execCtx.Status
	execCtx.Status = status.Status
	ea.mu.Unlock()
	
	// Handle status changes
	if status.Status != previousStatus {
		ea.logger.Info("Order status changed",
			ifs.Field{Key: "broker_order_id", Value: brokerOrderID},
			ifs.Field{Key: "old_status", Value: string(previousStatus)},
			ifs.Field{Key: "new_status", Value: string(status.Status)},
		)
		
		switch status.Status {
		case entities.OrderStatusExecuted:
			if status.ExecutedPrice != nil && status.ExecutedQty != nil {
				ea.publishExecutedOrderFromStatus(ctx, execCtx.Order, brokerOrderID, status)
			}
			
			// Remove from tracking
			ea.mu.Lock()
			delete(ea.orderTracker, brokerOrderID)
			ea.mu.Unlock()
			
		case entities.OrderStatusCancelled, entities.OrderStatusRejected:
			ea.publishOrderEvent(ctx, "order.cancelled", execCtx.Order, status, nil)
			
			// Remove from tracking
			ea.mu.Lock()
			delete(ea.orderTracker, brokerOrderID)
			ea.mu.Unlock()
		}
	}
	
	return nil
}

// publishExecutedOrder publishes an executed order event
func (ea *ExecutionAgent) publishExecutedOrder(ctx context.Context, order *entities.Order, 
	result *interfaces.OrderResult) {
	
	message := ExecutedOrderMessage{
		OrderID:       string(order.ID),
		BrokerOrderID: result.BrokerOrderID,
		Symbol:        string(order.Symbol),
		Side:          string(order.Side),
		Quantity:      order.Quantity,
		ExecutedPrice: *result.ExecutedPrice,
		ExecutedQty:   *result.ExecutedQty,
		Fees:          result.Fees,
		ExecutedAt:    result.Timestamp,
		BrokerName:    ea.trader.GetBrokerName(),
	}
	
	if err := ea.messageBus.Publish(ctx, "order.executed", message); err != nil {
		ea.logger.Error("Failed to publish executed order",
			ifs.Field{Key: "order_id", Value: string(order.ID)},
			ifs.Field{Key: "error", Value: err.Error()},
		)
	} else {
		ea.logger.Info("Published executed order",
			ifs.Field{Key: "order_id", Value: string(order.ID)},
			ifs.Field{Key: "broker_order_id", Value: result.BrokerOrderID},
		)
		
		ea.metrics.IncrementCounter("execution_agent_orders_executed", map[string]string{
			"symbol": string(order.Symbol),
			"side":   string(order.Side),
			"broker": ea.trader.GetBrokerName(),
		})
	}
}

// publishExecutedOrderFromStatus publishes an executed order event from status check
func (ea *ExecutionAgent) publishExecutedOrderFromStatus(ctx context.Context, order *entities.Order,
	brokerOrderID string, status *interfaces.OrderStatus) {
	
	message := ExecutedOrderMessage{
		OrderID:       string(order.ID),
		BrokerOrderID: brokerOrderID,
		Symbol:        string(order.Symbol),
		Side:          string(order.Side),
		Quantity:      order.Quantity,
		ExecutedPrice: *status.ExecutedPrice,
		ExecutedQty:   *status.ExecutedQty,
		Fees:          status.Fees,
		ExecutedAt:    status.LastUpdate,
		BrokerName:    ea.trader.GetBrokerName(),
	}
	
	if err := ea.messageBus.Publish(ctx, "order.executed", message); err != nil {
		ea.logger.Error("Failed to publish executed order from status",
			ifs.Field{Key: "order_id", Value: string(order.ID)},
			ifs.Field{Key: "error", Value: err.Error()},
		)
	} else {
		ea.metrics.IncrementCounter("execution_agent_orders_executed", map[string]string{
			"symbol": string(order.Symbol),
			"side":   string(order.Side),
			"broker": ea.trader.GetBrokerName(),
		})
	}
}

// publishOrderEvent publishes a generic order event
func (ea *ExecutionAgent) publishOrderEvent(ctx context.Context, topic string, order *entities.Order,
	status *interfaces.OrderStatus, err error) {
	
	event := map[string]interface{}{
		"order_id": string(order.ID),
		"symbol":   string(order.Symbol),
		"side":     string(order.Side),
		"quantity": order.Quantity,
		"broker":   ea.trader.GetBrokerName(),
	}
	
	if status != nil {
		event["broker_order_id"] = status.BrokerOrderID
		event["status"] = string(status.Status)
		event["timestamp"] = status.LastUpdate
	}
	
	if err != nil {
		event["error"] = err.Error()
	}
	
	if publishErr := ea.messageBus.Publish(ctx, topic, event); publishErr != nil {
		ea.logger.Error("Failed to publish order event",
			ifs.Field{Key: "topic", Value: topic},
			ifs.Field{Key: "order_id", Value: string(order.ID)},
			ifs.Field{Key: "error", Value: publishErr.Error()},
		)
	}
}

// calculateRetryDelay calculates the delay for retry attempts using exponential backoff
func (ea *ExecutionAgent) calculateRetryDelay(attempt int) time.Duration {
	delay := float64(ea.retryConfig.InitialDelay) * 
		(ea.retryConfig.BackoffFactor * float64(attempt))
	
	if delay > float64(ea.retryConfig.MaxDelay) {
		delay = float64(ea.retryConfig.MaxDelay)
	}
	
	return time.Duration(delay)
}

// isRetryableError determines if an error is retryable
func (ea *ExecutionAgent) isRetryableError(err error) bool {
	if brokerErr, ok := err.(*interfaces.BrokerError); ok {
		switch brokerErr.Code {
		case "CONNECTION_FAILED", "TIMEOUT", "TEMPORARY_ERROR":
			return true
		case "ORDER_REJECTED", "INSUFFICIENT_FUNDS", "INVALID_SYMBOL":
			return false
		default:
			return true
		}
	}
	
	// Default to retryable for unknown errors
	return true
}