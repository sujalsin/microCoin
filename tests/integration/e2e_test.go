//go:build integration

package integration

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"microcoin/internal/auth"
	"microcoin/internal/database"
	"microcoin/internal/idempotency"
	"microcoin/internal/ledger"
	"microcoin/internal/models"
	"microcoin/internal/orders"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestE2EFlow(t *testing.T) {
	// Setup test containers
	ctx := context.Background()

	// Start PostgreSQL
	postgresContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:16"),
		postgres.WithDatabase("microcoin"),
		postgres.WithUsername("microcoin"),
		postgres.WithPassword("password"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second)),
	)
	require.NoError(t, err)
	defer postgresContainer.Terminate(ctx)

	// Start Redis
	redisContainer, err := redis.RunContainer(ctx,
		testcontainers.WithImage("redis:7-alpine"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("Ready to accept connections").
				WithStartupTimeout(30*time.Second)),
	)
	require.NoError(t, err)
	defer redisContainer.Terminate(ctx)

	// Get connection strings
	postgresConnStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	redisConnStr, err := redisContainer.ConnectionString(ctx)
	require.NoError(t, err)

	// Connect to database
	db, err := sql.Open("postgres", postgresConnStr)
	require.NoError(t, err)
	defer db.Close()

	// Run migrations (simplified for test)
	err = runMigrations(db)
	require.NoError(t, err)

	// Test the complete flow
	t.Run("Complete Trading Flow", func(t *testing.T) {
		// 1. Sign up a user
		user, err := signupUser(db)
		require.NoError(t, err)
		require.NotNil(t, user)

		// 2. Top up account
		ledgerService := ledger.NewService(db)
		account, err := ledgerService.TopUpUser(user.ID, decimal.NewFromFloat(1000.0))
		require.NoError(t, err)
		assert.True(t, account.BalanceAvailable.Equal(decimal.NewFromFloat(1000.0)))

		// 3. Create a limit buy order
		orderService := orders.NewService(db, nil) // No quotes service for this test
		orderReq := &models.CreateOrderRequest{
			Symbol: models.SymbolBTCUSD,
			Side:   models.OrderSideBuy,
			Type:   models.OrderTypeLimit,
			Price:  &[]decimal.Decimal{decimal.NewFromFloat(50000.0)}[0],
			Qty:    decimal.NewFromFloat(0.01),
		}

		orderResp, err := orderService.CreateOrder(user.ID, orderReq)
		require.NoError(t, err)
		assert.Equal(t, models.OrderStatusNew, orderResp.Status)
		assert.True(t, orderResp.FilledQty.IsZero())

		// 4. Verify order was created
		order, err := orderService.GetOrder(uuid.MustParse(orderResp.OrderID))
		require.NoError(t, err)
		assert.Equal(t, user.ID, order.UserID)
		assert.Equal(t, models.SymbolBTCUSD, order.Symbol)
		assert.Equal(t, models.OrderSideBuy, order.Side)

		// 5. Check that funds were held
		accountRepo := database.NewAccountRepository(db)
		usdAccount, err := accountRepo.GetAccountByUserIDAndCurrency(user.ID, models.CurrencyUSD)
		require.NoError(t, err)

		expectedHold := decimal.NewFromFloat(500.0) // 0.01 * 50000
		assert.True(t, usdAccount.BalanceHold.Equal(expectedHold))
		assert.True(t, usdAccount.BalanceAvailable.Equal(decimal.NewFromFloat(500.0))) // 1000 - 500
	})

	t.Run("Idempotency Test", func(t *testing.T) {
		// Create a user
		user, err := signupUser(db)
		require.NoError(t, err)

		// Top up account
		ledgerService := ledger.NewService(db)
		_, err = ledgerService.TopUpUser(user.ID, decimal.NewFromFloat(1000.0))
		require.NoError(t, err)

		// Test idempotency service
		idempotencyService := idempotency.NewService(db)
		idemKey := uuid.New().String()
		fingerprint := "test-fingerprint"

		// First request
		tx, err := db.Begin()
		require.NoError(t, err)

		err = idempotencyService.StoreIdempotency(tx, user.ID, idemKey, fingerprint, 200, []byte(`{"success": true}`))
		require.NoError(t, err)

		err = tx.Commit()
		require.NoError(t, err)

		// Second request with same key and fingerprint
		existingKey, err := idempotencyService.CheckIdempotency(user.ID, idemKey, fingerprint)
		require.NoError(t, err)
		require.NotNil(t, existingKey)
		assert.Equal(t, 200, existingKey.ResponseCode)

		// Third request with different fingerprint (should fail)
		_, err = idempotencyService.CheckIdempotency(user.ID, idemKey, "different-fingerprint")
		assert.Error(t, err)
	})
}

func signupUser(db *sql.DB) (*models.User, error) {
	email := fmt.Sprintf("test-%s@example.com", uuid.New().String())
	password := "testpassword123"

	// Hash password
	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		return nil, err
	}

	// Create user
	userRepo := database.NewUserRepository(db)
	user, err := userRepo.CreateUser(email, passwordHash)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func runMigrations(db *sql.DB) error {
	// Simplified migration for testing
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TYPE currency AS ENUM ('USD', 'BTC', 'ETH')`,
		`CREATE TABLE IF NOT EXISTS accounts (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id),
			currency currency NOT NULL,
			balance_available NUMERIC(30,10) NOT NULL DEFAULT 0,
			balance_hold NUMERIC(30,10) NOT NULL DEFAULT 0,
			UNIQUE (user_id, currency)
		)`,
		`CREATE TABLE IF NOT EXISTS ledger_entries (
			id BIGSERIAL PRIMARY KEY,
			journal_id UUID NOT NULL,
			account_id UUID NOT NULL REFERENCES accounts(id),
			amount NUMERIC(30,10) NOT NULL,
			currency currency NOT NULL,
			ref_type TEXT NOT NULL,
			ref_id UUID NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TYPE order_side AS ENUM ('BUY','SELL')`,
		`CREATE TYPE order_type AS ENUM ('MARKET','LIMIT')`,
		`CREATE TYPE order_status AS ENUM ('NEW','PARTIALLY_FILLED','FILLED','CANCELED','REJECTED')`,
		`CREATE TABLE IF NOT EXISTS orders (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id),
			symbol TEXT NOT NULL,
			side order_side NOT NULL,
			type order_type NOT NULL,
			price NUMERIC(30,10),
			qty NUMERIC(30,10) NOT NULL,
			filled_qty NUMERIC(30,10) NOT NULL DEFAULT 0,
			status order_status NOT NULL DEFAULT 'NEW',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS idempotency_keys (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id),
			idem_key TEXT NOT NULL,
			request_fingerprint TEXT NOT NULL,
			response_code INT NOT NULL,
			response_body JSONB NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE (user_id, idem_key)
		)`,
		`CREATE OR REPLACE FUNCTION create_user_accounts()
		RETURNS TRIGGER AS $$
		BEGIN
			INSERT INTO accounts (user_id, currency) VALUES
				(NEW.id, 'USD'),
				(NEW.id, 'BTC'),
				(NEW.id, 'ETH');
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql`,
		`DROP TRIGGER IF EXISTS create_user_accounts_trigger ON users`,
		`CREATE TRIGGER create_user_accounts_trigger
			AFTER INSERT ON users
			FOR EACH ROW
			EXECUTE FUNCTION create_user_accounts()`,
	}

	for _, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			return fmt.Errorf("failed to run migration: %w", err)
		}
	}

	return nil
}
