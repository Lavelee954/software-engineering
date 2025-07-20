package entities

import "errors"

var (
	ErrOrderNotFound         = errors.New("order not found")
	ErrPositionNotFound      = errors.New("position not found")
	ErrInsufficientQuantity  = errors.New("insufficient quantity")
	ErrInsufficientCash      = errors.New("insufficient cash balance")
	ErrInvalidOrderType      = errors.New("invalid order type")
	ErrInvalidOrderSide      = errors.New("invalid order side")
	ErrOrderAlreadyExecuted  = errors.New("order already executed")
	ErrOrderAlreadyCancelled = errors.New("order already cancelled")
	ErrRiskLimitExceeded     = errors.New("risk limit exceeded")
	ErrPositionSizeExceeded  = errors.New("position size limit exceeded")
	ErrMarketClosed          = errors.New("market is closed")
	ErrInvalidSymbol         = errors.New("invalid symbol")
	ErrConnectionFailed      = errors.New("connection failed")
	ErrAuthenticationFailed  = errors.New("authentication failed")
	ErrRateLimitExceeded     = errors.New("rate limit exceeded")
)