package database

import (
	"database/sql"
	"fmt"

	"microcoin/internal/models"

	"github.com/google/uuid"
)

// OrderRepository handles order database operations
type OrderRepository struct {
	db *sql.DB
}

// NewOrderRepository creates a new order repository
func NewOrderRepository(db *sql.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

// CreateOrder creates a new order
func (r *OrderRepository) CreateOrder(order *models.Order) error {
	query := `
		INSERT INTO orders (id, user_id, symbol, side, type, price, qty, filled_qty, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := r.db.Exec(query,
		order.ID,
		order.UserID,
		order.Symbol,
		order.Side,
		order.Type,
		order.Price,
		order.Qty,
		order.FilledQty,
		order.Status,
		order.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create order: %w", err)
	}

	return nil
}

// GetOrderByID retrieves an order by ID
func (r *OrderRepository) GetOrderByID(id uuid.UUID) (*models.Order, error) {
	query := `
		SELECT id, user_id, symbol, side, type, price, qty, filled_qty, status, created_at
		FROM orders
		WHERE id = $1`

	var order models.Order
	err := r.db.QueryRow(query, id).Scan(
		&order.ID,
		&order.UserID,
		&order.Symbol,
		&order.Side,
		&order.Type,
		&order.Price,
		&order.Qty,
		&order.FilledQty,
		&order.Status,
		&order.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("order not found")
		}
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	return &order, nil
}

// GetOrdersByUserID retrieves orders for a user
func (r *OrderRepository) GetOrdersByUserID(userID uuid.UUID, limit, offset int) ([]models.Order, error) {
	query := `
		SELECT id, user_id, symbol, side, type, price, qty, filled_qty, status, created_at
		FROM orders
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders: %w", err)
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var order models.Order
		err := rows.Scan(
			&order.ID,
			&order.UserID,
			&order.Symbol,
			&order.Side,
			&order.Type,
			&order.Price,
			&order.Qty,
			&order.FilledQty,
			&order.Status,
			&order.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}
		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating orders: %w", err)
	}

	return orders, nil
}

// UpdateOrder updates an order
func (r *OrderRepository) UpdateOrder(tx *sql.Tx, order *models.Order) error {
	query := `
		UPDATE orders
		SET filled_qty = $1, status = $2
		WHERE id = $3`

	_, err := tx.Exec(query, order.FilledQty, order.Status, order.ID)
	if err != nil {
		return fmt.Errorf("failed to update order: %w", err)
	}

	return nil
}

// GetActiveOrdersBySymbol retrieves active orders for a symbol
func (r *OrderRepository) GetActiveOrdersBySymbol(symbol models.Symbol) ([]models.Order, error) {
	query := `
		SELECT id, user_id, symbol, side, type, price, qty, filled_qty, status, created_at
		FROM orders
		WHERE symbol = $1 AND status IN ('NEW', 'PARTIALLY_FILLED')
		ORDER BY created_at ASC`

	rows, err := r.db.Query(query, symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get active orders: %w", err)
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var order models.Order
		err := rows.Scan(
			&order.ID,
			&order.UserID,
			&order.Symbol,
			&order.Side,
			&order.Type,
			&order.Price,
			&order.Qty,
			&order.FilledQty,
			&order.Status,
			&order.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}
		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating orders: %w", err)
	}

	return orders, nil
}
