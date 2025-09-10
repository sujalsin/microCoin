package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"microcoin/internal/auth"
	"microcoin/internal/database"
	"microcoin/internal/idempotency"
	"microcoin/internal/ledger"
	"microcoin/internal/models"
	"microcoin/internal/orders"
	"microcoin/internal/quotes"
	"microcoin/internal/rate"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
)

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins in development
		},
	}
)

func main() {
	// Initialize database
	db, err := database.Connect(database.DefaultConfig())
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close(db)

	// Initialize Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer redisClient.Close()

	// Test Redis connection
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis connection failed: %v", err)
		redisClient = nil // Disable Redis features
	}

	// Initialize services
	quotesService := quotes.NewService(redisClient)
	orderService := orders.NewService(db, quotesService)
	ledgerService := ledger.NewService(db)
	idempotencyService := idempotency.NewService(db)

	// Initialize rate limiter
	var rateLimiter *rate.Limiter
	if redisClient != nil {
		rateLimiter = rate.NewLimiter(redisClient, 60, time.Minute)
	}

	// Start quotes service
	if err := quotesService.Start(ctx); err != nil {
		log.Fatalf("Failed to start quotes service: %v", err)
	}

	// Setup HTTP server
	router := mux.NewRouter()

	// Middleware
	router.Use(auth.AuthMiddleware)
	if rateLimiter != nil {
		router.Use(rate.RateLimitMiddleware(rateLimiter))
	}
	router.Use(corsMiddleware)
	router.Use(loggingMiddleware)

	// Health check
	router.HandleFunc("/health", healthHandler).Methods("GET")

	// Auth routes
	authRouter := router.PathPrefix("/auth").Subrouter()
	authRouter.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth for auth endpoints
			next.ServeHTTP(w, r)
		})
	})
	authRouter.HandleFunc("/signup", signupHandler(db)).Methods("POST")
	authRouter.HandleFunc("/login", loginHandler(db)).Methods("POST")

	// Protected routes
	apiRouter := router.PathPrefix("/api").Subrouter()
	apiRouter.HandleFunc("/fund/topup", topupHandler(db, ledgerService, idempotencyService)).Methods("POST")
	apiRouter.HandleFunc("/quotes", quotesHandler(quotesService)).Methods("GET")
	apiRouter.HandleFunc("/orders", createOrderHandler(db, orderService, idempotencyService)).Methods("POST")
	apiRouter.HandleFunc("/orders/{id}", getOrderHandler(orderService)).Methods("GET")
	apiRouter.HandleFunc("/portfolio", portfolioHandler(db, orderService)).Methods("GET")

	// WebSocket routes
	router.HandleFunc("/ws/quotes", websocketQuotesHandler(quotesService))

	// Start server
	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	// Start server in goroutine
	go func() {
		log.Println("Server starting on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

// Middleware
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Idempotency-Key")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
	})
}

// Handlers
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func signupHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req models.AuthRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Hash password
		passwordHash, err := auth.HashPassword(req.Password)
		if err != nil {
			http.Error(w, "Failed to hash password", http.StatusInternalServerError)
			return
		}

		// Create user
		userRepo := database.NewUserRepository(db)
		user, err := userRepo.CreateUser(req.Email, passwordHash)
		if err != nil {
			http.Error(w, "Failed to create user", http.StatusInternalServerError)
			return
		}

		// Generate tokens
		accessToken, refreshToken, err := auth.GenerateTokens(user.ID, user.Email)
		if err != nil {
			http.Error(w, "Failed to generate tokens", http.StatusInternalServerError)
			return
		}

		response := models.AuthResponse{
			Token:        accessToken,
			RefreshToken: refreshToken,
			User:         *user,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

func loginHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req models.AuthRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Get user
		userRepo := database.NewUserRepository(db)
		user, err := userRepo.GetUserByEmail(req.Email)
		if err != nil {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		// Verify password
		valid, err := auth.VerifyPassword(req.Password, user.PasswordHash)
		if err != nil || !valid {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		// Generate tokens
		accessToken, refreshToken, err := auth.GenerateTokens(user.ID, user.Email)
		if err != nil {
			http.Error(w, "Failed to generate tokens", http.StatusInternalServerError)
			return
		}

		response := models.AuthResponse{
			Token:        accessToken,
			RefreshToken: refreshToken,
			User:         *user,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

func topupHandler(db *sql.DB, ledgerService *ledger.Service, idempotencyService *idempotency.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user ID from context
		userID, ok := auth.GetUserIDFromContext(r.Context())
		if !ok {
			http.Error(w, "User not authenticated", http.StatusUnauthorized)
			return
		}

		// Get idempotency key
		idemKey := r.Header.Get("Idempotency-Key")
		if idemKey == "" {
			http.Error(w, "Idempotency-Key header required", http.StatusBadRequest)
			return
		}

		// Read request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}

		// Generate fingerprint
		headers := make(map[string]string)
		for key, values := range r.Header {
			if len(values) > 0 {
				headers[key] = values[0]
			}
		}
		fingerprint := idempotencyService.GenerateFingerprint(body, headers)

		// Check idempotency
		existingKey, err := idempotencyService.CheckIdempotency(userID, idemKey, fingerprint)
		if err != nil {
			http.Error(w, "Idempotency key mismatch", http.StatusConflict)
			return
		}

		// If we have a cached response, return it
		if existingKey != nil {
			w.WriteHeader(existingKey.ResponseCode)
			w.Write(existingKey.ResponseBody)
			return
		}

		// Parse request
		var req models.TopUpRequest
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Process top-up
		account, err := ledgerService.TopUpUser(userID, req.Amount)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to top up: %v", err), http.StatusInternalServerError)
			return
		}

		// Create response
		response := models.TopUpResponse{
			Balance: account.BalanceAvailable,
		}

		responseBody, err := json.Marshal(response)
		if err != nil {
			http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
			return
		}

		// Store idempotency key
		tx, err := db.Begin()
		if err != nil {
			http.Error(w, "Failed to begin transaction", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		if err := idempotencyService.StoreIdempotency(tx, userID, idemKey, fingerprint, http.StatusOK, responseBody); err != nil {
			http.Error(w, "Failed to store idempotency key", http.StatusInternalServerError)
			return
		}

		if err := tx.Commit(); err != nil {
			http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(responseBody)
	}
}

func quotesHandler(quotesService *quotes.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		symbol := r.URL.Query().Get("symbol")
		if symbol == "" {
			http.Error(w, "Symbol parameter required", http.StatusBadRequest)
			return
		}

		quote, err := quotesService.GetQuote(models.Symbol(symbol))
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get quote: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(quote)
	}
}

func createOrderHandler(db *sql.DB, orderService *orders.Service, idempotencyService *idempotency.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user ID from context
		userID, ok := auth.GetUserIDFromContext(r.Context())
		if !ok {
			http.Error(w, "User not authenticated", http.StatusUnauthorized)
			return
		}

		// Get idempotency key
		idemKey := r.Header.Get("Idempotency-Key")
		if idemKey == "" {
			http.Error(w, "Idempotency-Key header required", http.StatusBadRequest)
			return
		}

		// Read request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}

		// Generate fingerprint
		headers := make(map[string]string)
		for key, values := range r.Header {
			if len(values) > 0 {
				headers[key] = values[0]
			}
		}
		fingerprint := idempotencyService.GenerateFingerprint(body, headers)

		// Check idempotency
		existingKey, err := idempotencyService.CheckIdempotency(userID, idemKey, fingerprint)
		if err != nil {
			http.Error(w, "Idempotency key mismatch", http.StatusConflict)
			return
		}

		// If we have a cached response, return it
		if existingKey != nil {
			w.WriteHeader(existingKey.ResponseCode)
			w.Write(existingKey.ResponseBody)
			return
		}

		// Parse request
		var req models.CreateOrderRequest
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Create order
		response, err := orderService.CreateOrder(userID, &req)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to create order: %v", err), http.StatusInternalServerError)
			return
		}

		responseBody, err := json.Marshal(response)
		if err != nil {
			http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
			return
		}

		// Store idempotency key
		tx, err := db.Begin()
		if err != nil {
			http.Error(w, "Failed to begin transaction", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		if err := idempotencyService.StoreIdempotency(tx, userID, idemKey, fingerprint, http.StatusOK, responseBody); err != nil {
			http.Error(w, "Failed to store idempotency key", http.StatusInternalServerError)
			return
		}

		if err := tx.Commit(); err != nil {
			http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(responseBody)
	}
}

func getOrderHandler(orderService *orders.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		orderID, err := uuid.Parse(vars["id"])
		if err != nil {
			http.Error(w, "Invalid order ID", http.StatusBadRequest)
			return
		}

		order, err := orderService.GetOrder(orderID)
		if err != nil {
			http.Error(w, "Order not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(order)
	}
}

func portfolioHandler(db *sql.DB, orderService *orders.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user ID from context
		userID, ok := auth.GetUserIDFromContext(r.Context())
		if !ok {
			http.Error(w, "User not authenticated", http.StatusUnauthorized)
			return
		}

		// Get accounts
		accountRepo := database.NewAccountRepository(db)
		accounts, err := accountRepo.GetAccountsByUserID(userID)
		if err != nil {
			http.Error(w, "Failed to get accounts", http.StatusInternalServerError)
			return
		}

		// Convert to portfolio format
		var balances []models.AccountBalance
		for _, account := range accounts {
			balance := models.AccountBalance{
				Currency:         account.Currency,
				BalanceAvailable: account.BalanceAvailable,
				BalanceHold:      account.BalanceHold,
				BalanceTotal:     account.BalanceAvailable.Add(account.BalanceHold),
			}
			balances = append(balances, balance)
		}

		portfolio := models.Portfolio{
			Balances: balances,
			Positions: []models.Position{}, // TODO: Calculate positions
			PnL: models.PnL{
				Realized:   decimal.Zero,
				Unrealized: decimal.Zero,
				Total:      decimal.Zero,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(portfolio)
	}
}

func websocketQuotesHandler(quotesService *quotes.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("Failed to upgrade connection: %v", err)
			return
		}
		defer conn.Close()

		// Subscribe to all symbols
		btcCh := quotesService.Subscribe(models.SymbolBTCUSD)
		ethCh := quotesService.Subscribe(models.SymbolETHUSD)

		defer quotesService.Unsubscribe(models.SymbolBTCUSD, btcCh)
		defer quotesService.Unsubscribe(models.SymbolETHUSD, ethCh)

		// Send quotes to client
		for {
			select {
			case quote := <-btcCh:
				if err := conn.WriteJSON(quote); err != nil {
					log.Printf("Failed to write BTC quote: %v", err)
					return
				}
			case quote := <-ethCh:
				if err := conn.WriteJSON(quote); err != nil {
					log.Printf("Failed to write ETH quote: %v", err)
					return
				}
			}
		}
	}
}
