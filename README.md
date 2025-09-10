# MicroCoin - Paper Trading Crypto Backend

A small, dockerized backend that lets users:
- Sign up / log in (JWT)
- Top-up "paper USD"
- Stream real-time BTC/ETH quotes
- Place market/limit orders
- Maintain balances in a double-entry ledger

## 🏗️ Architecture

### Phase 1: Monolith (Current)
Single Go binary with internal packages for auth, ledger, limitbook, quotes, orders, idempotency, and rate limiting.

### Phase 2: Microservices (Week 3)
Split into 3 services:
- **Gateway**: REST + WS, auth, rate limiting
- **Quotes**: WS client → normalized ticks via Redis Pub/Sub
- **TradeLedger**: orders, limit book, fills, ledger, idempotency & outbox

## 🛠️ Tech Stack
- Go 1.21
- PostgreSQL 16
- Redis 7
- Docker Compose
- JWT for auth
- Argon2id for password hashing
- WebSocket for real-time quotes
- k6 for load testing

## 🚀 Quick Start

1. **Setup development environment:**
```bash
make dev-setup
```

2. **Run the monolith:**
```bash
make run
```

3. **Run the demo:**
```bash
./demo.sh
```

4. **Run tests:**
```bash
make test
```

## 📡 API Endpoints

### Auth
- `POST /auth/signup` - Register new user
- `POST /auth/login` - Login user

### Funding
- `POST /api/fund/topup` - Add paper USD (requires Idempotency-Key header)

### Quotes
- `GET /api/quotes?symbol=BTC-USD` - Get current quote
- `WS /ws/quotes` - Stream real-time quotes

### Orders
- `POST /api/orders` - Place order (requires Idempotency-Key header)
- `GET /api/orders/:id` - Get order details
- `GET /api/portfolio` - Get user portfolio

## 🗄️ Data Model

### Users & Auth
- Users table with email/password_hash
- JWT tokens (access + refresh)
- Argon2id password hashing

### Double-Entry Ledger
- Accounts table (USD, BTC, ETH)
- Ledger entries with journal_id for atomic operations
- Balance tracking (available + hold)

### Orders
- Support for MARKET and LIMIT orders
- Price-time priority matching
- Order status tracking

## 🧪 Testing

- **Unit tests**: Core business logic
- **Integration tests**: End-to-end flows with Testcontainers
- **Load tests**: k6 scripts for performance validation
- **Property tests**: Ledger balance invariants

### Running Tests
```bash
# Unit tests
make test

# Integration tests
make integration-test

# Load tests
make load-test

# Test coverage
make test-coverage
```

## 📊 Key Features

### ✅ Implemented
- [x] User authentication with JWT
- [x] Double-entry ledger system
- [x] Real-time quotes (mock data)
- [x] Limit order book with price-time priority
- [x] Idempotency for money operations
- [x] Rate limiting with Redis
- [x] WebSocket real-time quotes
- [x] Comprehensive testing suite
- [x] Docker Compose setup

### 🔄 In Progress
- [ ] Microservices split
- [ ] Real market data integration
- [ ] Advanced order types
- [ ] Performance optimization

## 🎯 Development Phases

### Week 1-2: Monolith MVP ✅
- [x] Project setup
- [x] Auth system
- [x] Basic ledger
- [x] Market orders
- [x] Real-time quotes
- [x] Testing suite

### Week 3: Service Split
- [ ] Extract quotes service
- [ ] Extract trade ledger service
- [ ] Add Redis Pub/Sub
- [ ] Outbox pattern

### Week 4: Polish
- [ ] Load testing
- [ ] Performance optimization
- [ ] Documentation
- [ ] Demo preparation

## 🔧 Configuration

### Environment Variables
- `DATABASE_URL` - PostgreSQL connection string
- `REDIS_URL` - Redis connection string
- `JWT_SECRET` - JWT signing secret

### Database
The system uses PostgreSQL with the following key tables:
- `users` - User accounts
- `accounts` - Currency balances per user
- `ledger_entries` - Double-entry bookkeeping
- `orders` - Trading orders
- `idempotency_keys` - Request deduplication

## 📈 Performance

The system is designed to handle:
- 100+ RPS for order placement
- Sub-500ms response times (p95)
- Real-time quote streaming
- Concurrent user sessions

## 🔒 Security

- Argon2id password hashing
- JWT token authentication
- Rate limiting per user
- Idempotency for financial operations
- Input validation and sanitization

## 🐳 Docker

```bash
# Build and run with Docker Compose
docker-compose up --build

# Run in development mode
make dev-setup
```

## 📝 API Examples

### Sign Up
```bash
curl -X POST http://localhost:8080/auth/signup \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password123"}'
```

### Top Up Account
```bash
curl -X POST http://localhost:8080/api/fund/topup \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Idempotency-Key: unique-key-123" \
  -d '{"amount":"1000.00"}'
```

### Place Order
```bash
curl -X POST http://localhost:8080/api/orders \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Idempotency-Key: order-key-456" \
  -d '{"symbol":"BTC-USD","side":"BUY","type":"LIMIT","price":"50000","qty":"0.01"}'
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## 📄 License

This project is licensed under the MIT License.
