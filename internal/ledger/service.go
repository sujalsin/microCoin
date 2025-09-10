package ledger

import (
	"database/sql"
	"fmt"

	"microcoin/internal/database"
	"microcoin/internal/models"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Service handles ledger business logic
type Service struct {
	db              *sql.DB
	ledgerRepo      *LedgerRepository
	accountRepo     *database.AccountRepository
}

// NewService creates a new ledger service
func NewService(db *sql.DB) *Service {
	return &Service{
		db:          db,
		ledgerRepo:  NewLedgerRepository(db),
		accountRepo: database.NewAccountRepository(db),
	}
}

// TopUpUser adds funds to a user's USD account
func (s *Service) TopUpUser(userID uuid.UUID, amount decimal.Decimal) (*models.Account, error) {
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("amount must be positive")
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get user's USD account
	account, err := s.accountRepo.GetAccountByUserIDAndCurrency(userID, models.CurrencyUSD)
	if err != nil {
		return nil, fmt.Errorf("failed to get USD account: %w", err)
	}

	// Create journal entries
	journalID := uuid.New()
	entries := []models.LedgerEntry{
		{
			JournalID: journalID,
			AccountID: account.ID,
			Amount:    amount, // Credit user's USD account
			Currency:  models.CurrencyUSD,
			RefType:   "TOPUP",
			RefID:     journalID,
		},
		{
			JournalID: journalID,
			AccountID: uuid.Nil, // System equity account (placeholder)
			Amount:    amount.Neg(), // Debit system equity
			Currency:  models.CurrencyUSD,
			RefType:   "TOPUP",
			RefID:     journalID,
		},
	}

	// Create journal
	if err := s.ledgerRepo.CreateJournal(tx, entries); err != nil {
		return nil, fmt.Errorf("failed to create journal: %w", err)
	}

	// Update account balance
	newBalance := account.BalanceAvailable.Add(amount)
	if err := s.accountRepo.UpdateAccountBalance(tx, account.ID, newBalance, account.BalanceHold); err != nil {
		return nil, fmt.Errorf("failed to update account balance: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Return updated account
	account.BalanceAvailable = newBalance
	return account, nil
}

// HoldFunds places a hold on funds for an order
func (s *Service) HoldFunds(userID uuid.UUID, currency models.Currency, amount decimal.Decimal) error {
	if amount.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("amount must be positive")
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get user's account
	account, err := s.accountRepo.GetAccountByUserIDAndCurrency(userID, currency)
	if err != nil {
		return fmt.Errorf("failed to get account: %w", err)
	}

	// Check if sufficient funds are available
	if account.BalanceAvailable.LessThan(amount) {
		return fmt.Errorf("insufficient funds: available=%s, required=%s", 
			account.BalanceAvailable.String(), amount.String())
	}

	// Update balances: move from available to hold
	newAvailable := account.BalanceAvailable.Sub(amount)
	newHold := account.BalanceHold.Add(amount)

	if err := s.accountRepo.UpdateAccountBalance(tx, account.ID, newAvailable, newHold); err != nil {
		return fmt.Errorf("failed to update account balance: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ReleaseHold releases held funds back to available
func (s *Service) ReleaseHold(userID uuid.UUID, currency models.Currency, amount decimal.Decimal) error {
	if amount.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("amount must be positive")
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get user's account
	account, err := s.accountRepo.GetAccountByUserIDAndCurrency(userID, currency)
	if err != nil {
		return fmt.Errorf("failed to get account: %w", err)
	}

	// Check if sufficient funds are held
	if account.BalanceHold.LessThan(amount) {
		return fmt.Errorf("insufficient held funds: held=%s, required=%s", 
			account.BalanceHold.String(), amount.String())
	}

	// Update balances: move from hold to available
	newAvailable := account.BalanceAvailable.Add(amount)
	newHold := account.BalanceHold.Sub(amount)

	if err := s.accountRepo.UpdateAccountBalance(tx, account.ID, newAvailable, newHold); err != nil {
		return fmt.Errorf("failed to update account balance: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// TransferFunds transfers funds between accounts (for trades)
func (s *Service) TransferFunds(fromAccountID, toAccountID uuid.UUID, amount decimal.Decimal, currency models.Currency, refType string, refID uuid.UUID) error {
	if amount.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("amount must be positive")
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get accounts
	fromAccount, err := s.accountRepo.GetAccountByID(fromAccountID)
	if err != nil {
		return fmt.Errorf("failed to get from account: %w", err)
	}

	toAccount, err := s.accountRepo.GetAccountByID(toAccountID)
	if err != nil {
		return fmt.Errorf("failed to get to account: %w", err)
	}

	// Check if sufficient funds are available in from account
	if fromAccount.BalanceAvailable.LessThan(amount) {
		return fmt.Errorf("insufficient funds in from account: available=%s, required=%s", 
			fromAccount.BalanceAvailable.String(), amount.String())
	}

	// Create journal entries
	journalID := uuid.New()
	entries := []models.LedgerEntry{
		{
			JournalID: journalID,
			AccountID: fromAccountID,
			Amount:    amount.Neg(), // Debit from account
			Currency:  currency,
			RefType:   refType,
			RefID:     refID,
		},
		{
			JournalID: journalID,
			AccountID: toAccountID,
			Amount:    amount, // Credit to account
			Currency:  currency,
			RefType:   refType,
			RefID:     refID,
		},
	}

	// Create journal
	if err := s.ledgerRepo.CreateJournal(tx, entries); err != nil {
		return fmt.Errorf("failed to create journal: %w", err)
	}

	// Update account balances
	fromNewBalance := fromAccount.BalanceAvailable.Sub(amount)
	toNewBalance := toAccount.BalanceAvailable.Add(amount)

	if err := s.accountRepo.UpdateAccountBalance(tx, fromAccountID, fromNewBalance, fromAccount.BalanceHold); err != nil {
		return fmt.Errorf("failed to update from account balance: %w", err)
	}

	if err := s.accountRepo.UpdateAccountBalance(tx, toAccountID, toNewBalance, toAccount.BalanceHold); err != nil {
		return fmt.Errorf("failed to update to account balance: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
