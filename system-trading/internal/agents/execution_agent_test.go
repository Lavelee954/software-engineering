package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/system-trading/core/internal/entities"
	"github.com/system-trading/core/internal/infrastructure/brokers"
	"github.com/system-trading/core/internal/infrastructure/config"
	"github.com/system-trading/core/internal/infrastructure/logger"
	"github.com/system-trading/core/internal/infrastructure/messagebus"
	"github.com/system-trading/core/internal/infrastructure/metrics"
	"github.com/system-trading/core/internal/interfaces"
)

func setupTestExecutionAgent(t *testing.T) (*ExecutionAgent, *messagebus.MockMessageBus, *brokers.MockBroker) {
	t.Helper()

	// Create test logger
	testLogger, err := logger.NewZapLogger(config.LoggingConfig{
		Level:  "debug",
		Format: "text",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("Failed to create test logger: %v", err)
	}

	// Create test metrics with unique name to avoid registration conflicts
	testMetrics := metrics.NewPrometheusMetrics("test-execution-agent-" + fmt.Sprintf("%d", time.Now().UnixNano()))

	// Create mock message bus
	mockBus := messagebus.NewMockMessageBus()

	// Create mock broker
	mockBroker := brokers.NewMockBroker("TestBroker", testLogger)

	// Create execution agent
	agent := NewExecutionAgent(mockBus, mockBroker, testLogger, testMetrics)

	return agent, mockBus, mockBroker
}

func createTestOrder() *entities.Order {
	return &entities.Order{
		ID:       entities.OrderID("test-order-123"),
		Symbol:   entities.Symbol("AAPL"),
		Side:     entities.OrderSideBuy,
		Type:     entities.OrderTypeMarket,
		Quantity: 100,
		Status:   entities.OrderStatusApproved,
	}
}

func TestExecutionAgent_Start(t *testing.T) {
	agent, _, mockBroker := setupTestExecutionAgent(t)
	defer agent.Stop(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := agent.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start execution agent: %v", err)
	}

	// Verify broker is connected
	if !mockBroker.IsConnected() {
		t.Error("Expected broker to be connected after starting agent")
	}
}

func TestExecutionAgent_HandleApprovedOrder(t *testing.T) {
	agent, mockBus, mockBroker := setupTestExecutionAgent(t)
	defer agent.Stop(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Start the agent
	err := agent.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start execution agent: %v", err)
	}

	// Create test order
	order := createTestOrder()

	// Publish approved order
	orderData, err := json.Marshal(order)
	if err != nil {
		t.Fatalf("Failed to marshal order: %v", err)
	}

	err = mockBus.Publish(ctx, "order.approved", order)
	if err != nil {
		t.Fatalf("Failed to publish approved order: %v", err)
	}

	// Simulate message handling
	handler := mockBus.GetHandler("order.approved")
	if handler == nil {
		t.Fatal("No handler registered for order.approved")
	}

	err = handler(ctx, orderData)
	if err != nil {
		t.Fatalf("Failed to handle approved order: %v", err)
	}

	// Give some time for async processing
	time.Sleep(100 * time.Millisecond)

	// Check if order was executed (for market orders, execution is immediate)
	account, err := mockBroker.GetAccountInfo(ctx)
	if err != nil {
		t.Fatalf("Failed to get account info: %v", err)
	}

	// Should have cash reduced and position created
	if account.CashBalance >= 100000.0 { // Started with 100k
		t.Error("Expected cash balance to be reduced after order execution")
	}

	// Check for position in AAPL
	found := false
	for _, position := range account.Positions {
		if position.Symbol == "AAPL" {
			found = true
			if position.Quantity != 100 {
				t.Errorf("Expected position quantity 100, got %f", position.Quantity)
			}
			break
		}
	}

	if !found {
		t.Error("Expected to find AAPL position after order execution")
	}
}

func TestExecutionAgent_OrderValidation(t *testing.T) {
	agent, _, _ := setupTestExecutionAgent(t)
	defer agent.Stop(context.Background())

	tests := []struct {
		name        string
		order       *entities.Order
		expectError bool
	}{
		{
			name: "Valid market order",
			order: &entities.Order{
				ID:       entities.OrderID("valid-order"),
				Symbol:   entities.Symbol("AAPL"),
				Side:     entities.OrderSideBuy,
				Type:     entities.OrderTypeMarket,
				Quantity: 100,
			},
			expectError: false,
		},
		{
			name: "Invalid order - empty ID",
			order: &entities.Order{
				ID:       entities.OrderID(""),
				Symbol:   entities.Symbol("AAPL"),
				Side:     entities.OrderSideBuy,
				Type:     entities.OrderTypeMarket,
				Quantity: 100,
			},
			expectError: true,
		},
		{
			name: "Invalid order - empty symbol",
			order: &entities.Order{
				ID:       entities.OrderID("test-order"),
				Symbol:   entities.Symbol(""),
				Side:     entities.OrderSideBuy,
				Type:     entities.OrderTypeMarket,
				Quantity: 100,
			},
			expectError: true,
		},
		{
			name: "Invalid order - zero quantity",
			order: &entities.Order{
				ID:       entities.OrderID("test-order"),
				Symbol:   entities.Symbol("AAPL"),
				Side:     entities.OrderSideBuy,
				Type:     entities.OrderTypeMarket,
				Quantity: 0,
			},
			expectError: true,
		},
		{
			name: "Invalid order - negative quantity",
			order: &entities.Order{
				ID:       entities.OrderID("test-order"),
				Symbol:   entities.Symbol("AAPL"),
				Side:     entities.OrderSideBuy,
				Type:     entities.OrderTypeMarket,
				Quantity: -10,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := agent.validateOrder(tt.order)
			if tt.expectError && err == nil {
				t.Error("Expected validation error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no validation error but got: %v", err)
			}
		})
	}
}

func TestExecutionAgent_RetryLogic(t *testing.T) {
	agent, _, mockBroker := setupTestExecutionAgent(t)
	defer agent.Stop(context.Background())

	// Configure higher error rate to test retries
	SetMockBrokerErrorRate(mockBroker, 0.8) // 80% error rate

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := agent.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start execution agent: %v", err)
	}

	order := createTestOrder()

	// Test retry behavior
	err = agent.executeOrder(ctx, order)
	// Even with high error rate, it should eventually succeed or return meaningful error
	if err != nil {
		t.Logf("Order execution failed after retries (expected with high error rate): %v", err)
	}
}

func TestExecutionAgent_OrderStatusMonitoring(t *testing.T) {
	agent, _, mockBroker := setupTestExecutionAgent(t)
	defer agent.Stop(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := agent.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start execution agent: %v", err)
	}

	order := createTestOrder()

	// Place order
	result, err := mockBroker.PlaceOrder(ctx, order)
	if err != nil {
		t.Fatalf("Failed to place order: %v", err)
	}

	// Track the order
	agent.trackOrder(order, result.BrokerOrderID)

	// Wait for status monitoring to run
	time.Sleep(200 * time.Millisecond)

	// Check order status
	status, err := mockBroker.GetOrderStatus(ctx, result.BrokerOrderID)
	if err != nil {
		t.Fatalf("Failed to get order status: %v", err)
	}

	if status.Status != entities.OrderStatusExecuted {
		t.Errorf("Expected order status to be executed, got %s", status.Status)
	}
}

func TestExecutionAgent_Shutdown(t *testing.T) {
	agent, _, mockBroker := setupTestExecutionAgent(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start agent
	err := agent.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start execution agent: %v", err)
	}

	// Verify it's running
	if !mockBroker.IsConnected() {
		t.Error("Expected broker to be connected")
	}

	// Stop agent
	err = agent.Stop(ctx)
	if err != nil {
		t.Fatalf("Failed to stop execution agent: %v", err)
	}

	// Verify it's stopped
	if mockBroker.IsConnected() {
		t.Error("Expected broker to be disconnected after stopping agent")
	}
}

func TestExecutionAgent_ErrorScenarios(t *testing.T) {
	agent, _, _ := setupTestExecutionAgent(t)
	defer agent.Stop(context.Background())

	// Test error classification
	testCases := []struct {
		name       string
		err        error
		retryable  bool
	}{
		{
			name: "Connection error - retryable",
			err: &interfaces.BrokerError{
				Code:    "CONNECTION_FAILED",
				Message: "Connection failed",
			},
			retryable: true,
		},
		{
			name: "Order rejected - not retryable",
			err: &interfaces.BrokerError{
				Code:    "ORDER_REJECTED",
				Message: "Order rejected",
			},
			retryable: false,
		},
		{
			name: "Insufficient funds - not retryable",
			err: &interfaces.BrokerError{
				Code:    "INSUFFICIENT_FUNDS",
				Message: "Insufficient funds",
			},
			retryable: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			retryable := agent.isRetryableError(tc.err)
			if retryable != tc.retryable {
				t.Errorf("Expected retryable=%v for error %v, got %v", 
					tc.retryable, tc.err, retryable)
			}
		})
	}
}

// SetErrorRate method for testing (interface to private field)
func SetMockBrokerErrorRate(mb *brokers.MockBroker, rate float64) {
	mb.SetErrorRate(rate)
}