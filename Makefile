.PHONY: build run test clean docker-build docker-up docker-down migrate-up migrate-down

# Build the application
build:
	go build -o bin/microcoin ./cmd/monolith

# Run the application
run: build
	./bin/microcoin

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

# Clean build artifacts
clean:
	rm -rf bin/

# Docker commands
docker-build:
	docker-compose build

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

# Database migrations
migrate-up:
	migrate -path migrations -database "postgres://microcoin:password@localhost:5432/microcoin?sslmode=disable" up

migrate-down:
	migrate -path migrations -database "postgres://microcoin:password@localhost:5432/microcoin?sslmode=disable" down

# Development setup
dev-setup: docker-up
	sleep 5
	make migrate-up

# Load testing
load-test:
	k6 run load-test/orders.js

# Integration tests
integration-test:
	go test -v -tags=integration ./tests/integration/...
