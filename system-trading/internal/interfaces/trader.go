package interfaces

import (
	"context"
	"time"

	"github.com/system-trading/core/internal/entities"
)

// Trader defines the interface for interacting with different brokerage APIs
// This abstraction allows the ExecutionAgent to work with multiple brokers
type Trader interface {
	// Connect establishes connection to the broker
	Connect(ctx context.Context) error
	
	// Disconnect closes the connection to the broker
	Disconnect(ctx context.Context) error
	
	// PlaceOrder submits an order to the broker
	PlaceOrder(ctx context.Context, order *entities.Order) (*OrderResult, error)
	
	// CancelOrder cancels an existing order
	CancelOrder(ctx context.Context, orderID string) error
	
	// GetOrderStatus retrieves the current status of an order
	GetOrderStatus(ctx context.Context, orderID string) (*OrderStatus, error)
	
	// GetAccountInfo retrieves account balance and positions
	GetAccountInfo(ctx context.Context) (*AccountInfo, error)
	
	// IsConnected returns true if connected to the broker
	IsConnected() bool
	
	// GetBrokerName returns the name of the broker
	GetBrokerName() string
}

// OrderResult represents the result of placing an order
type OrderResult struct {
	BrokerOrderID string                 `json:"broker_order_id"`
	Status        entities.OrderStatus   `json:"status"`
	Message       string                 `json:"message"`
	Timestamp     time.Time              `json:"timestamp"`
	Fees          float64                `json:"fees"`
	ExecutedPrice *float64               `json:"executed_price,omitempty"`
	ExecutedQty   *float64               `json:"executed_quantity,omitempty"`
}

// OrderStatus represents the current status of an order at the broker
type OrderStatus struct {
	BrokerOrderID   string                 `json:"broker_order_id"`
	Status          entities.OrderStatus   `json:"status"`
	ExecutedPrice   *float64               `json:"executed_price,omitempty"`
	ExecutedQty     *float64               `json:"executed_quantity,omitempty"`
	RemainingQty    *float64               `json:"remaining_quantity,omitempty"`
	Fees            float64                `json:"fees"`
	LastUpdate      time.Time              `json:"last_update"`
	Fills           []Fill                 `json:"fills,omitempty"`
}

// Fill represents a partial or complete fill of an order
type Fill struct {
	Price     float64   `json:"price"`
	Quantity  float64   `json:"quantity"`
	Fees      float64   `json:"fees"`
	Timestamp time.Time `json:"timestamp"`
}

// AccountInfo represents account balance and positions
type AccountInfo struct {
	AccountID     string              `json:"account_id"`
	CashBalance   float64             `json:"cash_balance"`
	TotalValue    float64             `json:"total_value"`
	BuyingPower   float64             `json:"buying_power"`
	Positions     []Position          `json:"positions"`
	LastUpdated   time.Time           `json:"last_updated"`
}

// Position represents a current position in a security
type Position struct {
	Symbol        string    `json:"symbol"`
	Quantity      float64   `json:"quantity"`
	AveragePrice  float64   `json:"average_price"`
	MarketValue   float64   `json:"market_value"`
	UnrealizedPnL float64   `json:"unrealized_pnl"`
	LastUpdated   time.Time `json:"last_updated"`
}

// BrokerError represents an error from a broker
type BrokerError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func (e *BrokerError) Error() string {
	if e.Details != "" {
		return e.Code + ": " + e.Message + " (" + e.Details + ")"
	}
	return e.Code + ": " + e.Message
}