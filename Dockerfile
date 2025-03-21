FROM golang:1.23.4-alpine AS builder

# Set the working directory
WORKDIR /app

# Install required packages
RUN apk add --no-cache git

# Copy go.mod and go.sum files first to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server/main.go

# Use a small alpine image for the final image
FROM alpine:latest

# Set the working directory
WORKDIR /app

# Install CA certificates for secure connections
RUN apk --no-cache add ca-certificates

# Copy the binary from the builder stage
COPY --from=builder /app/server .

# Copy migrations directory
COPY --from=builder /app/migrations ./migrations

# Expose the application port
EXPOSE 8080

# Start the application
CMD ["./server"]
