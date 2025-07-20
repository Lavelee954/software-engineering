package entities

import (
	"time"
)

type OrderID string
type Symbol string
type OrderType string
type OrderSide string
type OrderStatus string

const (
	OrderTypeMarket OrderType = "MARKET"
	OrderTypeLimit  OrderType = "LIMIT"
	OrderTypeStop   OrderType = "STOP"
)

const (
	OrderSideBuy  OrderSide = "BUY"
	OrderSideSell OrderSide = "SELL"
)

const (
	OrderStatusPending   OrderStatus = "PENDING"
	OrderStatusApproved  OrderStatus = "APPROVED"
	OrderStatusExecuted  OrderStatus = "EXECUTED"
	OrderStatusCancelled OrderStatus = "CANCELLED"
	OrderStatusRejected  OrderStatus = "REJECTED"
)

type Order struct {
	ID        OrderID     `json:"id"`
	Symbol    Symbol      `json:"symbol"`
	Side      OrderSide   `json:"side"`
	Type      OrderType   `json:"type"`
	Quantity  float64     `json:"quantity"`
	Price     *float64    `json:"price,omitempty"`
	Status    OrderStatus `json:"status"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
	ExecutedAt *time.Time  `json:"executed_at,omitempty"`
	ExecutedPrice *float64 `json:"executed_price,omitempty"`
	ExecutedQuantity *float64 `json:"executed_quantity,omitempty"`
}

func NewOrder(symbol Symbol, side OrderSide, orderType OrderType, quantity float64, price *float64) *Order {
	now := time.Now()
	return &Order{
		ID:        OrderID(generateID()),
		Symbol:    symbol,
		Side:      side,
		Type:      orderType,
		Quantity:  quantity,
		Price:     price,
		Status:    OrderStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func (o *Order) Approve() {
	o.Status = OrderStatusApproved
	o.UpdatedAt = time.Now()
}

func (o *Order) Execute(price float64, quantity float64) {
	now := time.Now()
	o.Status = OrderStatusExecuted
	o.ExecutedAt = &now
	o.ExecutedPrice = &price
	o.ExecutedQuantity = &quantity
	o.UpdatedAt = now
}

func (o *Order) Reject() {
	o.Status = OrderStatusRejected
	o.UpdatedAt = time.Now()
}

func (o *Order) Cancel() {
	o.Status = OrderStatusCancelled
	o.UpdatedAt = time.Now()
}

func generateID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}