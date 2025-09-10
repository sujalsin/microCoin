package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Currency represents supported currencies
type Currency string

const (
	CurrencyUSD Currency = "USD"
	CurrencyBTC Currency = "BTC"
	CurrencyETH Currency = "ETH"
)

// OrderSide represents buy or sell
type OrderSide string

const (
	OrderSideBuy  OrderSide = "BUY"
	OrderSideSell OrderSide = "SELL"
)

// OrderType represents market or limit orders
type OrderType string

const (
	OrderTypeMarket OrderType = "MARKET"
	OrderTypeLimit  OrderType = "LIMIT"
)

// OrderStatus represents order lifecycle states
type OrderStatus string

const (
	OrderStatusNew             OrderStatus = "NEW"
	OrderStatusPartiallyFilled OrderStatus = "PARTIALLY_FILLED"
	OrderStatusFilled          OrderStatus = "FILLED"
	OrderStatusCanceled        OrderStatus = "CANCELED"
	OrderStatusRejected        OrderStatus = "REJECTED"
)

// Symbol represents trading pairs
type Symbol string

const (
	SymbolBTCUSD Symbol = "BTC-USD"
	SymbolETHUSD Symbol = "ETH-USD"
)

// User represents a user account
type User struct {
	ID           uuid.UUID `json:"id" db:"id"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"-" db:"password_hash"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// Account represents a user's account for a specific currency
type Account struct {
	ID               uuid.UUID     `json:"id" db:"id"`
	UserID           uuid.UUID     `json:"user_id" db:"user_id"`
	Currency         Currency      `json:"currency" db:"currency"`
	BalanceAvailable decimal.Decimal `json:"balance_available" db:"balance_available"`
	BalanceHold      decimal.Decimal `json:"balance_hold" db:"balance_hold"`
}

// LedgerEntry represents a single entry in the double-entry ledger
type LedgerEntry struct {
	ID        int64           `json:"id" db:"id"`
	JournalID uuid.UUID       `json:"journal_id" db:"journal_id"`
	AccountID uuid.UUID       `json:"account_id" db:"account_id"`
	Amount    decimal.Decimal `json:"amount" db:"amount"`
	Currency  Currency        `json:"currency" db:"currency"`
	RefType   string          `json:"ref_type" db:"ref_type"`
	RefID     uuid.UUID       `json:"ref_id" db:"ref_id"`
	CreatedAt time.Time       `json:"created_at" db:"created_at"`
}

// Order represents a trading order
type Order struct {
	ID        uuid.UUID       `json:"id" db:"id"`
	UserID    uuid.UUID       `json:"user_id" db:"user_id"`
	Symbol    Symbol          `json:"symbol" db:"symbol"`
	Side      OrderSide       `json:"side" db:"side"`
	Type      OrderType       `json:"type" db:"type"`
	Price     *decimal.Decimal `json:"price,omitempty" db:"price"`
	Qty       decimal.Decimal `json:"qty" db:"qty"`
	FilledQty decimal.Decimal `json:"filled_qty" db:"filled_qty"`
	Status    OrderStatus     `json:"status" db:"status"`
	CreatedAt time.Time       `json:"created_at" db:"created_at"`
}

// IdempotencyKey represents an idempotency key for request deduplication
type IdempotencyKey struct {
	ID                  uuid.UUID `json:"id" db:"id"`
	UserID              uuid.UUID `json:"user_id" db:"user_id"`
	IdemKey             string    `json:"idem_key" db:"idem_key"`
	RequestFingerprint  string    `json:"request_fingerprint" db:"request_fingerprint"`
	ResponseCode        int       `json:"response_code" db:"response_code"`
	ResponseBody        []byte    `json:"response_body" db:"response_body"`
	CreatedAt           time.Time `json:"created_at" db:"created_at"`
}

// OutboxEvent represents an event to be published
type OutboxEvent struct {
	ID          int64     `json:"id" db:"id"`
	Topic       string    `json:"topic" db:"topic"`
	Payload     []byte    `json:"payload" db:"payload"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	PublishedAt *time.Time `json:"published_at,omitempty" db:"published_at"`
}

// Quote represents a market quote
type Quote struct {
	Symbol Symbol          `json:"symbol"`
	Bid    decimal.Decimal `json:"bid"`
	Ask    decimal.Decimal `json:"ask"`
	TS     time.Time       `json:"ts"`
}

// Trade represents a completed trade
type Trade struct {
	ID        uuid.UUID       `json:"id"`
	Symbol    Symbol          `json:"symbol"`
	Side      OrderSide       `json:"side"`
	Price     decimal.Decimal `json:"price"`
	Qty       decimal.Decimal `json:"qty"`
	TakerID   uuid.UUID       `json:"taker_id"`
	MakerID   uuid.UUID       `json:"maker_id"`
	CreatedAt time.Time       `json:"created_at"`
}

// Portfolio represents a user's portfolio
type Portfolio struct {
	Balances []AccountBalance `json:"balances"`
	Positions []Position       `json:"positions"`
	PnL      PnL              `json:"pnl"`
}

// AccountBalance represents a balance for a specific currency
type AccountBalance struct {
	Currency         Currency        `json:"currency"`
	BalanceAvailable decimal.Decimal `json:"balance_available"`
	BalanceHold      decimal.Decimal `json:"balance_hold"`
	BalanceTotal     decimal.Decimal `json:"balance_total"`
}

// Position represents a trading position
type Position struct {
	Symbol    Symbol          `json:"symbol"`
	Qty       decimal.Decimal `json:"qty"`
	AvgPrice  decimal.Decimal `json:"avg_price"`
	UnrealizedPnL decimal.Decimal `json:"unrealized_pnl"`
}

// PnL represents profit and loss
type PnL struct {
	Realized   decimal.Decimal `json:"realized"`
	Unrealized decimal.Decimal `json:"unrealized"`
	Total      decimal.Decimal `json:"total"`
}
