# MicroCoin Migration Plan: Monolith → Microservices

## Overview

This document outlines the plan to split the MicroCoin monolith into 3 microservices as outlined in the original specification.

## Current State (Monolith)

The monolith is a single Go binary with the following internal packages:
- `auth` - JWT authentication and password hashing
- `ledger` - Double-entry bookkeeping system
- `limitbook` - Order book and matching engine
- `quotes` - Real-time market data (mock)
- `orders` - Order management
- `idempotency` - Request deduplication
- `rate` - Rate limiting with Redis

## Target Architecture (Microservices)

### 1. Gateway Service
**Responsibilities:**
- REST API endpoints
- WebSocket connections
- Authentication middleware
- Rate limiting
- Idempotency enforcement
- Request routing to other services

**Endpoints:**
- `POST /auth/signup` - User registration
- `POST /auth/login` - User authentication
- `POST /api/fund/topup` - Account funding
- `GET /api/quotes` - Quote snapshots
- `POST /api/orders` - Order placement
- `GET /api/orders/:id` - Order details
- `GET /api/portfolio` - User portfolio
- `WS /ws/quotes` - Real-time quotes

**Dependencies:**
- PostgreSQL (for user data and idempotency)
- Redis (for rate limiting)
- TradeLedger service (for orders and portfolio)
- Quotes service (for real-time data)

### 2. Quotes Service
**Responsibilities:**
- Connect to external market data feeds
- Normalize quote data
- Publish to Redis Pub/Sub
- Maintain quote snapshots

**Data Flow:**
```
External Feed → Quotes Service → Redis Pub/Sub → Gateway → WebSocket Clients
```

**Dependencies:**
- Redis (for publishing quotes)
- External market data APIs

### 3. TradeLedger Service
**Responsibilities:**
- Order management and matching
- Limit order book maintenance
- Trade execution
- Double-entry ledger operations
- Outbox pattern for event publishing

**Data Flow:**
```
Gateway → TradeLedger → PostgreSQL → Outbox → Redis Pub/Sub
```

**Dependencies:**
- PostgreSQL (for orders, ledger, outbox)
- Redis (for consuming quotes, publishing trades)

## Migration Strategy

### Phase 1: Extract Quotes Service
1. Create new `cmd/quotes` binary
2. Move quotes logic to separate service
3. Implement Redis Pub/Sub for quote publishing
4. Update Gateway to consume from Redis
5. Test quote flow end-to-end

### Phase 2: Extract TradeLedger Service
1. Create new `cmd/tradeledger` binary
2. Move order and ledger logic to separate service
3. Implement outbox pattern for trade events
4. Update Gateway to route orders to TradeLedger
5. Test order flow end-to-end

### Phase 3: Refactor Gateway
1. Remove extracted logic from Gateway
2. Implement service-to-service communication
3. Add circuit breakers and retries
4. Update Docker Compose configuration
5. End-to-end testing

## Service Communication

### Gateway ↔ TradeLedger
- **Protocol:** HTTP/gRPC
- **Authentication:** Service-to-service JWT
- **Endpoints:**
  - `POST /orders` - Create order
  - `GET /orders/:id` - Get order
  - `GET /portfolio/:user_id` - Get portfolio

### Gateway ↔ Quotes Service
- **Protocol:** Redis Pub/Sub
- **Channels:**
  - `quotes:BTC-USD`
  - `quotes:ETH-USD`

### TradeLedger ↔ Quotes Service
- **Protocol:** Redis Pub/Sub
- **Channels:**
  - `quotes:BTC-USD` (consume)
  - `quotes:ETH-USD` (consume)
  - `trades:BTC-USD` (publish)
  - `trades:ETH-USD` (publish)

## Data Consistency

### Idempotency
- Gateway maintains idempotency keys
- TradeLedger validates idempotency for financial operations
- Redis used for distributed idempotency checks

### Eventual Consistency
- Trade events published via outbox pattern
- Gateway subscribes to trade events for real-time updates
- Database transactions ensure consistency within services

## Deployment

### Docker Compose
```yaml
services:
  postgres:
    # Database for all services
  
  redis:
    # Message broker and caching
  
  gateway:
    # API gateway and WebSocket server
  
  quotes:
    # Market data service
  
  trade-ledger:
    # Order management and matching
```

### Environment Variables
- `GATEWAY_PORT=8080`
- `QUOTES_PORT=8081`
- `TRADE_LEDGER_PORT=8082`
- `POSTGRES_URL=postgres://...`
- `REDIS_URL=redis://...`

## Testing Strategy

### Unit Tests
- Each service tested independently
- Mock external dependencies
- Test business logic in isolation

### Integration Tests
- Test service-to-service communication
- Test Redis Pub/Sub flows
- Test database transactions

### End-to-End Tests
- Full user journey testing
- Load testing across services
- Failure scenario testing

## Rollback Plan

1. **Immediate:** Revert to monolith deployment
2. **Data:** Ensure database schema compatibility
3. **Configuration:** Update environment variables
4. **Monitoring:** Verify all metrics and logs

## Success Criteria

- [ ] All existing functionality preserved
- [ ] Response times within 10% of monolith
- [ ] 99.9% uptime during migration
- [ ] Zero data loss
- [ ] All tests passing
- [ ] Load testing successful

## Timeline

- **Week 3, Day 1-2:** Extract Quotes Service
- **Week 3, Day 3-4:** Extract TradeLedger Service
- **Week 3, Day 5:** Refactor Gateway and testing
- **Week 4:** Performance optimization and documentation

## Risks and Mitigation

### Risk: Service Communication Latency
**Mitigation:** Implement connection pooling and caching

### Risk: Data Consistency Issues
**Mitigation:** Use database transactions and outbox pattern

### Risk: Service Discovery
**Mitigation:** Use Docker Compose networking and environment variables

### Risk: Monitoring Complexity
**Mitigation:** Implement structured logging and health checks
