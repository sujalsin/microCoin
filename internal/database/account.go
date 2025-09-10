package database

import (
	"database/sql"
	"fmt"

	"microcoin/internal/models"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// AccountRepository handles account database operations
type AccountRepository struct {
	db *sql.DB
}

// NewAccountRepository creates a new account repository
func NewAccountRepository(db *sql.DB) *AccountRepository {
	return &AccountRepository{db: db}
}

// GetAccountByUserIDAndCurrency retrieves an account by user ID and currency
func (r *AccountRepository) GetAccountByUserIDAndCurrency(userID uuid.UUID, currency models.Currency) (*models.Account, error) {
	query := `
		SELECT id, user_id, currency, balance_available, balance_hold
		FROM accounts
		WHERE user_id = $1 AND currency = $2`

	var account models.Account
	err := r.db.QueryRow(query, userID, currency).Scan(
		&account.ID,
		&account.UserID,
		&account.Currency,
		&account.BalanceAvailable,
		&account.BalanceHold,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("account not found")
		}
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	return &account, nil
}

// GetAccountsByUserID retrieves all accounts for a user
func (r *AccountRepository) GetAccountsByUserID(userID uuid.UUID) ([]models.Account, error) {
	query := `
		SELECT id, user_id, currency, balance_available, balance_hold
		FROM accounts
		WHERE user_id = $1
		ORDER BY currency`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get accounts: %w", err)
	}
	defer rows.Close()

	var accounts []models.Account
	for rows.Next() {
		var account models.Account
		err := rows.Scan(
			&account.ID,
			&account.UserID,
			&account.Currency,
			&account.BalanceAvailable,
			&account.BalanceHold,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan account: %w", err)
		}
		accounts = append(accounts, account)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating accounts: %w", err)
	}

	return accounts, nil
}

// UpdateAccountBalance updates account balances within a transaction
func (r *AccountRepository) UpdateAccountBalance(tx *sql.Tx, accountID uuid.UUID, available, hold decimal.Decimal) error {
	query := `
		UPDATE accounts
		SET balance_available = $1, balance_hold = $2
		WHERE id = $3`

	_, err := tx.Exec(query, available, hold, accountID)
	if err != nil {
		return fmt.Errorf("failed to update account balance: %w", err)
	}

	return nil
}

// GetAccountByID retrieves an account by ID
func (r *AccountRepository) GetAccountByID(id uuid.UUID) (*models.Account, error) {
	query := `
		SELECT id, user_id, currency, balance_available, balance_hold
		FROM accounts
		WHERE id = $1`

	var account models.Account
	err := r.db.QueryRow(query, id).Scan(
		&account.ID,
		&account.UserID,
		&account.Currency,
		&account.BalanceAvailable,
		&account.BalanceHold,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("account not found")
		}
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	return &account, nil
}
