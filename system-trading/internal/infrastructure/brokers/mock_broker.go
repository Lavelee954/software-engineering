package brokers

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/system-trading/core/internal/entities"
	"github.com/system-trading/core/internal/interfaces"
	ifs "github.com/system-trading/core/internal/usecases/interfaces"
)

// MockBroker is a simulated broker for testing and development
type MockBroker struct {
	name        string
	connected   bool
	orders      map[string]*MockOrder
	account     *interfaces.AccountInfo
	latency     time.Duration
	errorRate   float64
	mu          sync.RWMutex
	logger      ifs.Logger
}

// MockOrder represents an order in the mock broker
type MockOrder struct {
	Order         *entities.Order
	BrokerOrderID string
	Status        entities.OrderStatus
	Fills         []interfaces.Fill
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// NewMockBroker creates a new mock broker instance
func NewMockBroker(name string, logger ifs.Logger) *MockBroker {
	return &MockBroker{
		name:      name,
		connected: false,
		orders:    make(map[string]*MockOrder),
		latency:   100 * time.Millisecond, // Simulate network latency
		errorRate: 0.01,                   // 1% error rate
		logger:    logger,
		account: &interfaces.AccountInfo{
			AccountID:   "MOCK_ACCOUNT_001",
			CashBalance: 100000.0, // $100k starting cash
			TotalValue:  100000.0,
			BuyingPower: 100000.0,
			Positions:   []interfaces.Position{},
			LastUpdated: time.Now(),
		},
	}
}

// Connect establishes connection to the mock broker
func (mb *MockBroker) Connect(ctx context.Context) error {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	
	// Simulate connection time
	select {
	case <-time.After(mb.latency):
	case <-ctx.Done():
		return ctx.Err()
	}
	
	// Simulate occasional connection failures
	if rand.Float64() < mb.errorRate {
		return &interfaces.BrokerError{
			Code:    "CONNECTION_FAILED",
			Message: "Failed to connect to mock broker",
		}
	}
	
	mb.connected = true
	mb.logger.Info("Connected to mock broker",
		ifs.Field{Key: "broker", Value: mb.name},
	)
	
	return nil
}

// Disconnect closes the connection to the mock broker
func (mb *MockBroker) Disconnect(ctx context.Context) error {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	
	mb.connected = false
	mb.logger.Info("Disconnected from mock broker",
		ifs.Field{Key: "broker", Value: mb.name},
	)
	
	return nil
}

// PlaceOrder submits an order to the mock broker
func (mb *MockBroker) PlaceOrder(ctx context.Context, order *entities.Order) (*interfaces.OrderResult, error) {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	
	if !mb.connected {
		return nil, &interfaces.BrokerError{
			Code:    "NOT_CONNECTED",
			Message: "Not connected to broker",
		}
	}
	
	// Simulate processing time
	select {
	case <-time.After(mb.latency):
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	
	// Simulate occasional order rejection
	if rand.Float64() < mb.errorRate {
		return nil, &interfaces.BrokerError{
			Code:    "ORDER_REJECTED",
			Message: "Order rejected by mock broker",
			Details: fmt.Sprintf("Simulated rejection for order %s", order.ID),
		}
	}
	
	// Generate broker order ID
	brokerOrderID := fmt.Sprintf("MOCK_%d", time.Now().UnixNano())
	
	// Create mock order
	mockOrder := &MockOrder{
		Order:         order,
		BrokerOrderID: brokerOrderID,
		Status:        entities.OrderStatusPending,
		Fills:         []interfaces.Fill{},
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	
	mb.orders[brokerOrderID] = mockOrder
	
	// For market orders, simulate immediate execution
	if order.Type == entities.OrderTypeMarket {
		go mb.simulateExecution(brokerOrderID)
	}
	
	result := &interfaces.OrderResult{
		BrokerOrderID: brokerOrderID,
		Status:        entities.OrderStatusPending,
		Message:       "Order submitted successfully",
		Timestamp:     time.Now(),
		Fees:          mb.calculateFees(order),
	}
	
	mb.logger.Info("Order placed with mock broker",
		ifs.Field{Key: "broker_order_id", Value: brokerOrderID},
		ifs.Field{Key: "symbol", Value: string(order.Symbol)},
		ifs.Field{Key: "side", Value: string(order.Side)},
		ifs.Field{Key: "quantity", Value: order.Quantity},
	)
	
	return result, nil
}

// CancelOrder cancels an existing order
func (mb *MockBroker) CancelOrder(ctx context.Context, orderID string) error {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	
	if !mb.connected {
		return &interfaces.BrokerError{
			Code:    "NOT_CONNECTED",
			Message: "Not connected to broker",
		}
	}
	
	mockOrder, exists := mb.orders[orderID]
	if !exists {
		return &interfaces.BrokerError{
			Code:    "ORDER_NOT_FOUND",
			Message: "Order not found",
		}
	}
	
	if mockOrder.Status == entities.OrderStatusExecuted {
		return &interfaces.BrokerError{
			Code:    "ORDER_ALREADY_EXECUTED",
			Message: "Cannot cancel executed order",
		}
	}
	
	mockOrder.Status = entities.OrderStatusCancelled
	mockOrder.UpdatedAt = time.Now()
	
	mb.logger.Info("Order cancelled",
		ifs.Field{Key: "broker_order_id", Value: orderID},
	)
	
	return nil
}

// GetOrderStatus retrieves the current status of an order
func (mb *MockBroker) GetOrderStatus(ctx context.Context, orderID string) (*interfaces.OrderStatus, error) {
	mb.mu.RLock()
	defer mb.mu.RUnlock()
	
	if !mb.connected {
		return nil, &interfaces.BrokerError{
			Code:    "NOT_CONNECTED",
			Message: "Not connected to broker",
		}
	}
	
	mockOrder, exists := mb.orders[orderID]
	if !exists {
		return nil, &interfaces.BrokerError{
			Code:    "ORDER_NOT_FOUND",
			Message: "Order not found",
		}
	}
	
	status := &interfaces.OrderStatus{
		BrokerOrderID: mockOrder.BrokerOrderID,
		Status:        mockOrder.Status,
		Fees:          mb.calculateFees(mockOrder.Order),
		LastUpdate:    mockOrder.UpdatedAt,
		Fills:         mockOrder.Fills,
	}
	
	// Add execution details if executed
	if mockOrder.Status == entities.OrderStatusExecuted && len(mockOrder.Fills) > 0 {
		totalQuantity := 0.0
		weightedPrice := 0.0
		
		for _, fill := range mockOrder.Fills {
			totalQuantity += fill.Quantity
			weightedPrice += fill.Price * fill.Quantity
		}
		
		if totalQuantity > 0 {
			avgPrice := weightedPrice / totalQuantity
			status.ExecutedPrice = &avgPrice
			status.ExecutedQty = &totalQuantity
			remaining := mockOrder.Order.Quantity - totalQuantity
			status.RemainingQty = &remaining
		}
	}
	
	return status, nil
}

// GetAccountInfo retrieves account balance and positions
func (mb *MockBroker) GetAccountInfo(ctx context.Context) (*interfaces.AccountInfo, error) {
	mb.mu.RLock()
	defer mb.mu.RUnlock()
	
	if !mb.connected {
		return nil, &interfaces.BrokerError{
			Code:    "NOT_CONNECTED",
			Message: "Not connected to broker",
		}
	}
	
	// Return a copy of account info
	accountCopy := *mb.account
	accountCopy.LastUpdated = time.Now()
	
	return &accountCopy, nil
}

// IsConnected returns true if connected to the broker
func (mb *MockBroker) IsConnected() bool {
	mb.mu.RLock()
	defer mb.mu.RUnlock()
	return mb.connected
}

// GetBrokerName returns the name of the broker
func (mb *MockBroker) GetBrokerName() string {
	return mb.name
}

// SetErrorRate allows tests to control the error rate for testing retry logic
func (mb *MockBroker) SetErrorRate(rate float64) {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	mb.errorRate = rate
}

// simulateExecution simulates order execution for market orders
func (mb *MockBroker) simulateExecution(brokerOrderID string) {
	// Wait for a random execution delay (50-500ms)
	delay := time.Duration(50+rand.Intn(450)) * time.Millisecond
	time.Sleep(delay)
	
	mb.mu.Lock()
	defer mb.mu.Unlock()
	
	mockOrder, exists := mb.orders[brokerOrderID]
	if !exists || mockOrder.Status != entities.OrderStatusPending {
		return
	}
	
	// Simulate market price with small random variation
	basePrice := 100.0 // Default price
	marketPrice := basePrice + (rand.Float64()-0.5)*2 // ±$1 variation
	
	// Create fill
	fill := interfaces.Fill{
		Price:     marketPrice,
		Quantity:  mockOrder.Order.Quantity,
		Fees:      mb.calculateFees(mockOrder.Order),
		Timestamp: time.Now(),
	}
	
	mockOrder.Fills = append(mockOrder.Fills, fill)
	mockOrder.Status = entities.OrderStatusExecuted
	mockOrder.UpdatedAt = time.Now()
	
	// Update account positions
	mb.updateAccountPosition(string(mockOrder.Order.Symbol), mockOrder.Order.Side, 
		mockOrder.Order.Quantity, marketPrice)
	
	mb.logger.Info("Mock order executed",
		ifs.Field{Key: "broker_order_id", Value: brokerOrderID},
		ifs.Field{Key: "price", Value: marketPrice},
		ifs.Field{Key: "quantity", Value: mockOrder.Order.Quantity},
	)
}

// calculateFees calculates commission fees for an order
func (mb *MockBroker) calculateFees(order *entities.Order) float64 {
	// Simple fee structure: $0.005 per share, minimum $1
	fees := order.Quantity * 0.005
	if fees < 1.0 {
		fees = 1.0
	}
	return fees
}

// updateAccountPosition updates account cash and positions after execution
func (mb *MockBroker) updateAccountPosition(symbol string, side entities.OrderSide, 
	quantity, price float64) {
	
	tradeValue := quantity * price
	fees := quantity * 0.005
	if fees < 1.0 {
		fees = 1.0
	}
	
	// Update cash balance
	if side == entities.OrderSideBuy {
		mb.account.CashBalance -= tradeValue + fees
	} else {
		mb.account.CashBalance += tradeValue - fees
	}
	
	// Update positions
	found := false
	for i := range mb.account.Positions {
		if mb.account.Positions[i].Symbol == symbol {
			found = true
			if side == entities.OrderSideBuy {
				// Calculate new average price
				oldValue := mb.account.Positions[i].Quantity * mb.account.Positions[i].AveragePrice
				newQuantity := mb.account.Positions[i].Quantity + quantity
				if newQuantity > 0 {
					mb.account.Positions[i].AveragePrice = (oldValue + tradeValue) / newQuantity
				}
				mb.account.Positions[i].Quantity = newQuantity
			} else {
				mb.account.Positions[i].Quantity -= quantity
			}
			
			mb.account.Positions[i].MarketValue = mb.account.Positions[i].Quantity * price
			mb.account.Positions[i].UnrealizedPnL = mb.account.Positions[i].MarketValue - 
				(mb.account.Positions[i].Quantity * mb.account.Positions[i].AveragePrice)
			mb.account.Positions[i].LastUpdated = time.Now()
			
			// Remove position if quantity is zero
			if mb.account.Positions[i].Quantity == 0 {
				mb.account.Positions = append(mb.account.Positions[:i], mb.account.Positions[i+1:]...)
			}
			break
		}
	}
	
	// Add new position if buying and position doesn't exist
	if !found && side == entities.OrderSideBuy {
		position := interfaces.Position{
			Symbol:        symbol,
			Quantity:      quantity,
			AveragePrice:  price,
			MarketValue:   tradeValue,
			UnrealizedPnL: 0,
			LastUpdated:   time.Now(),
		}
		mb.account.Positions = append(mb.account.Positions, position)
	}
	
	// Update total account value
	totalPositionValue := 0.0
	for _, pos := range mb.account.Positions {
		totalPositionValue += pos.MarketValue
	}
	mb.account.TotalValue = mb.account.CashBalance + totalPositionValue
	mb.account.BuyingPower = mb.account.CashBalance // Simplified
	mb.account.LastUpdated = time.Now()
}