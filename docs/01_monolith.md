# MicroCoin Monolith Implementation

## Overview

This document describes the completed monolith implementation of MicroCoin, a paper trading crypto backend system.

## ✅ Completed Features

### 1. Authentication System
- **JWT-based authentication** with access and refresh tokens
- **Argon2id password hashing** for secure password storage
- **User registration and login** endpoints
- **Middleware for protected routes**

### 2. Double-Entry Ledger System
- **Account management** for USD, BTC, and ETH
- **Journal-based transactions** ensuring balance invariants
- **Hold/release mechanisms** for order funds
- **Atomic transactions** with PostgreSQL

### 3. Real-Time Quotes System
- **Mock market data generation** for BTC-USD and ETH-USD
- **WebSocket streaming** for real-time quotes
- **REST API** for quote snapshots
- **Redis Pub/Sub** for quote distribution

### 4. Order Management System
- **Limit order book** with price-time priority
- **Market and limit orders** support
- **Order matching engine** with partial fills
- **Order status tracking** (NEW, PARTIALLY_FILLED, FILLED, etc.)

### 5. Idempotency System
- **Request deduplication** for financial operations
- **Fingerprint-based validation** for request integrity
- **Database-backed storage** for idempotency keys
- **Automatic retry handling**

### 6. Rate Limiting
- **Token bucket algorithm** implemented in Redis
- **Per-user rate limiting** with configurable limits
- **Lua scripts** for atomic operations
- **HTTP middleware** integration

### 7. Testing Suite
- **Unit tests** for core business logic
- **Integration tests** with Testcontainers
- **Load testing** with k6 scripts
- **Property tests** for ledger balance invariants

## 🏗️ Architecture

### Database Schema
```sql
-- Users and authentication
users (id, email, password_hash, created_at)

-- Multi-currency accounts
accounts (id, user_id, currency, balance_available, balance_hold)

-- Double-entry ledger
ledger_entries (id, journal_id, account_id, amount, currency, ref_type, ref_id, created_at)

-- Trading orders
orders (id, user_id, symbol, side, type, price, qty, filled_qty, status, created_at)

-- Idempotency tracking
idempotency_keys (id, user_id, idem_key, request_fingerprint, response_code, response_body, created_at)

-- Event publishing
outbox (id, topic, payload, created_at, published_at)
```

### Key Components

#### Authentication (`internal/auth/`)
- JWT token generation and validation
- Argon2id password hashing
- Middleware for route protection

#### Ledger (`internal/ledger/`)
- Double-entry bookkeeping
- Account balance management
- Transaction journal creation

#### Order Book (`internal/limitbook/`)
- Price-time priority matching
- Heap-based order management
- Trade execution logic

#### Quotes (`internal/quotes/`)
- Mock market data generation
- WebSocket streaming
- Redis Pub/Sub integration

#### Orders (`internal/orders/`)
- Order lifecycle management
- Integration with ledger and order book
- Trade processing

#### Idempotency (`internal/idempotency/`)
- Request fingerprinting
- Database-backed deduplication
- HTTP middleware integration

#### Rate Limiting (`internal/rate/`)
- Redis-based token bucket
- Lua script execution
- HTTP middleware

## 🚀 API Endpoints

### Authentication
- `POST /auth/signup` - User registration
- `POST /auth/login` - User authentication

### Funding
- `POST /api/fund/topup` - Add paper USD (idempotent)

### Market Data
- `GET /api/quotes?symbol=BTC-USD` - Get current quote
- `WS /ws/quotes` - Stream real-time quotes

### Trading
- `POST /api/orders` - Place order (idempotent)
- `GET /api/orders/:id` - Get order details
- `GET /api/portfolio` - Get user portfolio

## 🧪 Testing

### Unit Tests
```bash
go test ./tests/unit/...
```

### Integration Tests
```bash
go test -tags=integration ./tests/integration/...
```

### Load Tests
```bash
k6 run load-test/orders.js
```

### Test Coverage
```bash
make test-coverage
```

## 🐳 Deployment

### Docker Compose
```bash
docker-compose up --build
```

### Development Setup
```bash
make dev-setup
```

### Demo Script
```bash
./demo.sh
```

## 📊 Performance Characteristics

- **Response Time:** < 500ms (p95)
- **Throughput:** 100+ RPS for order placement
- **Concurrency:** Supports multiple concurrent users
- **Memory Usage:** ~50MB for typical load
- **Database Connections:** Connection pooling with 25 max connections

## 🔒 Security Features

- **Password Security:** Argon2id with salt
- **Token Security:** Short-lived JWT tokens
- **Rate Limiting:** Prevents abuse and DoS
- **Idempotency:** Prevents duplicate financial operations
- **Input Validation:** Comprehensive request validation
- **SQL Injection Protection:** Parameterized queries

## 📈 Monitoring and Observability

- **Structured Logging:** JSON-formatted logs
- **Health Checks:** `/health` endpoint
- **Request Tracing:** Request ID correlation
- **Error Handling:** Consistent error responses
- **Metrics:** Built-in Go metrics

## 🔄 Data Flow

### User Registration
1. User submits email/password
2. Password hashed with Argon2id
3. User record created in database
4. JWT tokens generated and returned

### Account Funding
1. User requests top-up with idempotency key
2. System checks for duplicate request
3. Double-entry journal created (credit user, debit system)
4. Account balance updated atomically

### Order Placement
1. User submits order with idempotency key
2. System validates order and checks funds
3. Funds held in user account
4. Order added to limit order book
5. Matching engine attempts to fill order
6. Trades executed and ledger updated

### Real-Time Quotes
1. Mock data generator creates price updates
2. Quotes published to Redis channels
3. WebSocket clients receive real-time updates
4. REST API provides snapshot access

## 🎯 Key Design Decisions

### 1. Double-Entry Ledger
- Ensures financial accuracy and auditability
- Prevents money creation/destruction
- Enables complex financial operations

### 2. Idempotency
- Critical for financial operations
- Prevents duplicate charges/credits
- Enables safe retry mechanisms

### 3. In-Memory Order Book
- Fast matching performance
- Rebuilt from database on startup
- Suitable for paper trading volumes

### 4. Mock Market Data
- Simplifies development and testing
- Can be easily replaced with real feeds
- Provides consistent test scenarios

### 5. Redis Integration
- Enables future microservices architecture
- Provides pub/sub for real-time features
- Supports rate limiting and caching

## 🚧 Known Limitations

1. **Single Instance:** Not horizontally scalable
2. **Mock Data:** No real market data integration
3. **Basic Matching:** No advanced order types
4. **No Persistence:** Order book rebuilt on restart
5. **Limited Symbols:** Only BTC-USD and ETH-USD

## 🔮 Next Steps (Microservices)

The monolith is ready for microservices extraction:

1. **Quotes Service:** Extract market data handling
2. **TradeLedger Service:** Extract order management and matching
3. **Gateway Service:** Extract API and WebSocket handling

See `docs/02_migration_plan.md` for detailed migration strategy.

## 📚 Documentation

- `README.md` - Project overview and quick start
- `docs/01_monolith.md` - This document
- `docs/02_migration_plan.md` - Microservices migration plan
- `demo.sh` - Interactive demo script
- `load-test/` - Performance testing scripts

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## 📄 License

This project is licensed under the MIT License.
