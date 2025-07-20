package entities

import (
	"time"
)

type PositionID string

type Position struct {
	ID            PositionID `json:"id"`
	Symbol        Symbol     `json:"symbol"`
	Quantity      float64    `json:"quantity"`
	AveragePrice  float64    `json:"average_price"`
	CurrentPrice  float64    `json:"current_price"`
	MarketValue   float64    `json:"market_value"`
	UnrealizedPnL float64    `json:"unrealized_pnl"`
	RealizedPnL   float64    `json:"realized_pnl"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type Portfolio struct {
	ID               string               `json:"id"`
	Cash             float64              `json:"cash"`
	TotalValue       float64              `json:"total_value"`
	Positions        map[Symbol]*Position `json:"positions"`
	TotalPnL         float64              `json:"total_pnl"`
	DayPnL           float64              `json:"day_pnl"`
	LastUpdated      time.Time            `json:"last_updated"`
}

func NewPortfolio(initialCash float64) *Portfolio {
	return &Portfolio{
		ID:          generateID(),
		Cash:        initialCash,
		TotalValue:  initialCash,
		Positions:   make(map[Symbol]*Position),
		TotalPnL:    0.0,
		DayPnL:      0.0,
		LastUpdated: time.Now(),
	}
}

func (p *Portfolio) AddPosition(symbol Symbol, quantity float64, price float64) {
	now := time.Now()
	
	if position, exists := p.Positions[symbol]; exists {
		newQuantity := position.Quantity + quantity
		newAveragePrice := ((position.AveragePrice * position.Quantity) + (price * quantity)) / newQuantity
		position.Quantity = newQuantity
		position.AveragePrice = newAveragePrice
		position.UpdatedAt = now
	} else {
		p.Positions[symbol] = &Position{
			ID:           PositionID(generateID()),
			Symbol:       symbol,
			Quantity:     quantity,
			AveragePrice: price,
			CurrentPrice: price,
			MarketValue:  quantity * price,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
	}
	
	p.Cash -= quantity * price
	p.updateTotalValue()
}

func (p *Portfolio) RemovePosition(symbol Symbol, quantity float64, price float64) error {
	position, exists := p.Positions[symbol]
	if !exists {
		return ErrPositionNotFound
	}
	
	if position.Quantity < quantity {
		return ErrInsufficientQuantity
	}
	
	realizedPnL := (price - position.AveragePrice) * quantity
	position.RealizedPnL += realizedPnL
	position.Quantity -= quantity
	position.UpdatedAt = time.Now()
	
	if position.Quantity == 0 {
		delete(p.Positions, symbol)
	}
	
	p.Cash += quantity * price
	p.TotalPnL += realizedPnL
	p.updateTotalValue()
	
	return nil
}

func (p *Portfolio) UpdatePositionPrice(symbol Symbol, price float64) {
	if position, exists := p.Positions[symbol]; exists {
		position.CurrentPrice = price
		position.MarketValue = position.Quantity * price
		position.UnrealizedPnL = (price - position.AveragePrice) * position.Quantity
		position.UpdatedAt = time.Now()
		p.updateTotalValue()
	}
}

func (p *Portfolio) updateTotalValue() {
	totalPositionValue := 0.0
	totalUnrealizedPnL := 0.0
	
	for _, position := range p.Positions {
		totalPositionValue += position.MarketValue
		totalUnrealizedPnL += position.UnrealizedPnL
	}
	
	p.TotalValue = p.Cash + totalPositionValue
	p.LastUpdated = time.Now()
}

func (p *Portfolio) GetPosition(symbol Symbol) (*Position, bool) {
	position, exists := p.Positions[symbol]
	return position, exists
}

func (p *Portfolio) GetCashBalance() float64 {
	return p.Cash
}

func (p *Portfolio) GetTotalValue() float64 {
	return p.TotalValue
}