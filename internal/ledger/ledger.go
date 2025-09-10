package ledger

import (
	"database/sql"
	"fmt"

	"microcoin/internal/models"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// LedgerRepository handles ledger database operations
type LedgerRepository struct {
	db *sql.DB
}

// NewLedgerRepository creates a new ledger repository
func NewLedgerRepository(db *sql.DB) *LedgerRepository {
	return &LedgerRepository{db: db}
}

// CreateJournal creates a balanced journal entry
func (r *LedgerRepository) CreateJournal(tx *sql.Tx, entries []models.LedgerEntry) error {
	if len(entries) == 0 {
		return fmt.Errorf("journal must have at least one entry")
	}

	// Validate that the journal is balanced (sum of amounts = 0)
	var total decimal.Decimal
	for _, entry := range entries {
		total = total.Add(entry.Amount)
	}

	if !total.IsZero() {
		return fmt.Errorf("journal is not balanced: total = %s", total.String())
	}

	// Insert all entries
	query := `
		INSERT INTO ledger_entries (journal_id, account_id, amount, currency, ref_type, ref_id)
		VALUES ($1, $2, $3, $4, $5, $6)`

	for _, entry := range entries {
		_, err := tx.Exec(query,
			entry.JournalID,
			entry.AccountID,
			entry.Amount,
			entry.Currency,
			entry.RefType,
			entry.RefID,
		)
		if err != nil {
			return fmt.Errorf("failed to create ledger entry: %w", err)
		}
	}

	return nil
}

// GetLedgerEntriesByJournalID retrieves all entries for a journal
func (r *LedgerRepository) GetLedgerEntriesByJournalID(journalID uuid.UUID) ([]models.LedgerEntry, error) {
	query := `
		SELECT id, journal_id, account_id, amount, currency, ref_type, ref_id, created_at
		FROM ledger_entries
		WHERE journal_id = $1
		ORDER BY created_at`

	rows, err := r.db.Query(query, journalID)
	if err != nil {
		return nil, fmt.Errorf("failed to get ledger entries: %w", err)
	}
	defer rows.Close()

	var entries []models.LedgerEntry
	for rows.Next() {
		var entry models.LedgerEntry
		err := rows.Scan(
			&entry.ID,
			&entry.JournalID,
			&entry.AccountID,
			&entry.Amount,
			&entry.Currency,
			&entry.RefType,
			&entry.RefID,
			&entry.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan ledger entry: %w", err)
		}
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating ledger entries: %w", err)
	}

	return entries, nil
}

// GetLedgerEntriesByAccountID retrieves ledger entries for an account
func (r *LedgerRepository) GetLedgerEntriesByAccountID(accountID uuid.UUID, limit, offset int) ([]models.LedgerEntry, error) {
	query := `
		SELECT id, journal_id, account_id, amount, currency, ref_type, ref_id, created_at
		FROM ledger_entries
		WHERE account_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(query, accountID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get ledger entries: %w", err)
	}
	defer rows.Close()

	var entries []models.LedgerEntry
	for rows.Next() {
		var entry models.LedgerEntry
		err := rows.Scan(
			&entry.ID,
			&entry.JournalID,
			&entry.AccountID,
			&entry.Amount,
			&entry.Amount,
			&entry.Currency,
			&entry.RefType,
			&entry.RefID,
			&entry.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan ledger entry: %w", err)
		}
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating ledger entries: %w", err)
	}

	return entries, nil
}

// ValidateJournalBalance validates that a journal is balanced
func (r *LedgerRepository) ValidateJournalBalance(journalID uuid.UUID) (bool, error) {
	query := `
		SELECT COALESCE(SUM(amount), 0)
		FROM ledger_entries
		WHERE journal_id = $1`

	var total decimal.Decimal
	err := r.db.QueryRow(query, journalID).Scan(&total)
	if err != nil {
		return false, fmt.Errorf("failed to validate journal balance: %w", err)
	}

	return total.IsZero(), nil
}
