# MicroCoin - Paper Trading Crypto Backend

A comprehensive paper trading platform that lets users:
- Sign up / log in (JWT)
- Top-up "paper USD"
- Stream real-time BTC/ETH quotes
- Place market/limit orders
- Maintain balances in a double-entry ledger

## 🏗️ Architecture

A well-structured Go application with clean separation of concerns:
- **Authentication**: JWT-based auth with Argon2id password hashing
- **Trading Engine**: Limit order book with price-time priority matching
- **Ledger System**: Double-entry bookkeeping for financial accuracy
- **Real-time Data**: WebSocket streaming with Redis Pub/Sub
- **Security**: Rate limiting, idempotency, and input validation

## 🛠️ Tech Stack
- Go 1.23
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

2. **Run the application:**
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

### 🔄 Future Enhancements
- [ ] Real market data integration
- [ ] Advanced order types (stop-loss, take-profit)
- [ ] Portfolio analytics and reporting
- [ ] Mobile app integration
- [ ] Advanced charting and technical indicators

## 🎯 Development Roadmap

### Phase 1: Core Platform ✅
- [x] Project setup and architecture
- [x] Authentication and user management
- [x] Double-entry ledger system
- [x] Order management and matching
- [x] Real-time quotes and WebSocket
- [x] Comprehensive testing suite

### Phase 2: Enhanced Features
- [ ] Real market data feeds
- [ ] Advanced order types
- [ ] Portfolio analytics
- [ ] Performance optimizations

### Phase 3: Platform Extensions
- [ ] Mobile API endpoints
- [ ] Advanced charting
- [ ] Social trading features
- [ ] Risk management tools

## 🔧 Configuration

### Environment Variables
- `DATABASE_URL` - PostgreSQL connection string
- `REDIS_URL` - Redis connection string
- `JWT_SECRET` - JWT signing secret

### Database Schema
The system uses PostgreSQL with a well-designed schema:
- `users` - User accounts and authentication
- `accounts` - Multi-currency balance tracking
- `ledger_entries` - Double-entry bookkeeping for financial accuracy
- `orders` - Trading order management
- `idempotency_keys` - Request deduplication for safety

## 📈 Performance

The system is optimized for:
- 100+ RPS for order placement
- Sub-500ms response times (p95)
- Real-time quote streaming via WebSocket
- Concurrent user sessions with connection pooling
- Efficient in-memory order book matching

## 🔒 Security

- **Password Security**: Argon2id hashing with salt
- **Authentication**: JWT tokens with short expiration
- **Rate Limiting**: Per-user request throttling
- **Idempotency**: Prevents duplicate financial operations
- **Input Validation**: Comprehensive request sanitization
- **SQL Injection Protection**: Parameterized queries

## 🐳 Docker

```bash
# Build and run with Docker Compose
docker-compose up --build

# Run in development mode
make dev-setup
```

## 🏗️ Project Structure

```
microCoin/
├── cmd/monolith/          # Main application entry point
├── internal/              # Internal application packages
│   ├── auth/             # Authentication and JWT handling
│   ├── database/         # Database layer and repositories
│   ├── ledger/           # Double-entry bookkeeping system
│   ├── limitbook/        # Order book and matching engine
│   ├── quotes/           # Real-time market data
│   ├── orders/           # Order management and processing
│   ├── idempotency/      # Request deduplication
│   ├── rate/             # Rate limiting middleware
│   └── models/           # Data models and types
├── migrations/           # Database schema migrations
├── tests/                # Comprehensive test suites
├── load-test/            # Performance testing with k6
├── docs/                 # Documentation and guides
└── demo.sh              # Interactive demo script
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
