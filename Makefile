.PHONY: test build run clean docker-build docker-run

# Default target
all: test build

# Build the application
build:
	go build -o bin/server cmd/server/main.go

# Run the application
run:
	go run cmd/server/main.go

# Run tests
test:
	go test -v ./...

# Run only the multi-device tests
test-multidevice:
	go test -v ./internal/auth -run TestMultiDeviceLogout

# Run token expiration tests
test-expiration:
	go test -v ./internal/auth -run TestTokenExpirationInBlacklist

# Clean build artifacts
clean:
	rm -rf bin/

# Build Docker image
docker-build:
	docker build -t jwt-blacklist:latest .

# Run in Docker with Redis
docker-run:
	docker-compose up

# Start Redis for local development
redis-start:
	docker run --name redis-jwt -p 6379:6379 -d redis

# Stop Redis container
redis-stop:
	docker stop redis-jwt
	docker rm redis-jwt
