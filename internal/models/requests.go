package models

import (
	"github.com/shopspring/decimal"
)

// AuthRequest represents authentication request
type AuthRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

// AuthResponse represents authentication response
type AuthResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	User         User   `json:"user"`
}

// TopUpRequest represents a top-up request
type TopUpRequest struct {
	Amount decimal.Decimal `json:"amount" validate:"required,gt=0"`
}

// TopUpResponse represents a top-up response
type TopUpResponse struct {
	Balance decimal.Decimal `json:"balance"`
}

// CreateOrderRequest represents an order creation request
type CreateOrderRequest struct {
	Symbol Symbol           `json:"symbol" validate:"required"`
	Side   OrderSide        `json:"side" validate:"required"`
	Type   OrderType        `json:"type" validate:"required"`
	Price  *decimal.Decimal `json:"price,omitempty"`
	Qty    decimal.Decimal  `json:"qty" validate:"required,gt=0"`
}

// CreateOrderResponse represents an order creation response
type CreateOrderResponse struct {
	OrderID      string           `json:"order_id"`
	Status       OrderStatus      `json:"status"`
	FilledQty    decimal.Decimal  `json:"filled_qty"`
	AvgFillPrice *decimal.Decimal `json:"avg_fill_price,omitempty"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail represents error details
type ErrorDetail struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

// Error codes
const (
	ErrorCodeRateLimit         = "RATE_LIMIT"
	ErrorCodeBadRequest        = "BAD_REQUEST"
	ErrorCodeUnauthorized      = "UNAUTHORIZED"
	ErrorCodeForbidden         = "FORBIDDEN"
	ErrorCodeNotFound          = "NOT_FOUND"
	ErrorCodeInsufficientFunds = "INSUFFICIENT_FUNDS"
	ErrorCodeIdemMismatch      = "IDEM_MISMATCH"
	ErrorCodeInternalError     = "INTERNAL_ERROR"
	ErrorCodeInvalidSymbol     = "INVALID_SYMBOL"
	ErrorCodeInvalidOrderType  = "INVALID_ORDER_TYPE"
	ErrorCodeOrderNotFound     = "ORDER_NOT_FOUND"
)
