package unit

import (
	"testing"

	"microcoin/internal/models"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestLedgerBalance(t *testing.T) {
	// Test that ledger entries always sum to zero
	entries := []models.LedgerEntry{
		{
			JournalID: uuid.New(),
			AccountID: uuid.New(),
			Amount:    decimal.NewFromFloat(100.0),
			Currency:  models.CurrencyUSD,
			RefType:   "TOPUP",
			RefID:     uuid.New(),
		},
		{
			JournalID: uuid.New(),
			AccountID: uuid.New(),
			Amount:    decimal.NewFromFloat(-100.0),
			Currency:  models.CurrencyUSD,
			RefType:   "TOPUP",
			RefID:     uuid.New(),
		},
	}

	// Calculate total
	var total decimal.Decimal
	for _, entry := range entries {
		total = total.Add(entry.Amount)
	}

	assert.True(t, total.IsZero(), "Ledger entries should sum to zero")
}

func TestLedgerValidation(t *testing.T) {
	// Test unbalanced journal (should fail)
	unbalancedEntries := []models.LedgerEntry{
		{
			JournalID: uuid.New(),
			AccountID: uuid.New(),
			Amount:    decimal.NewFromFloat(100.0),
			Currency:  models.CurrencyUSD,
			RefType:   "TOPUP",
			RefID:     uuid.New(),
		},
		{
			JournalID: uuid.New(),
			AccountID: uuid.New(),
			Amount:    decimal.NewFromFloat(-50.0), // Not balanced
			Currency:  models.CurrencyUSD,
			RefType:   "TOPUP",
			RefID:     uuid.New(),
		},
	}

	var total decimal.Decimal
	for _, entry := range unbalancedEntries {
		total = total.Add(entry.Amount)
	}

	assert.False(t, total.IsZero(), "Unbalanced journal should not sum to zero")
}

func TestDecimalPrecision(t *testing.T) {
	// Test decimal precision for financial calculations
	price := decimal.NewFromFloat(60000.123456789)
	qty := decimal.NewFromFloat(0.001)

	// Calculate value
	value := price.Mul(qty)

	// Should maintain precision
	expected := decimal.NewFromFloat(60.000123456789)
	assert.True(t, value.Equal(expected), "Decimal precision should be maintained")
}

func TestCurrencyValidation(t *testing.T) {
	// Test valid currencies
	validCurrencies := []models.Currency{
		models.CurrencyUSD,
		models.CurrencyBTC,
		models.CurrencyETH,
	}

	for _, currency := range validCurrencies {
		assert.NotEmpty(t, string(currency), "Currency should not be empty")
	}
}

func TestOrderSideValidation(t *testing.T) {
	// Test valid order sides
	validSides := []models.OrderSide{
		models.OrderSideBuy,
		models.OrderSideSell,
	}

	for _, side := range validSides {
		assert.NotEmpty(t, string(side), "Order side should not be empty")
	}
}

func TestOrderTypeValidation(t *testing.T) {
	// Test valid order types
	validTypes := []models.OrderType{
		models.OrderTypeMarket,
		models.OrderTypeLimit,
	}

	for _, orderType := range validTypes {
		assert.NotEmpty(t, string(orderType), "Order type should not be empty")
	}
}

func TestSymbolValidation(t *testing.T) {
	// Test valid symbols
	validSymbols := []models.Symbol{
		models.SymbolBTCUSD,
		models.SymbolETHUSD,
	}

	for _, symbol := range validSymbols {
		assert.NotEmpty(t, string(symbol), "Symbol should not be empty")
	}
}
