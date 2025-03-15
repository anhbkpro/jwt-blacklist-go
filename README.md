# JWT Authentication with Token Blacklisting

A Go project demonstrating JWT authentication with token blacklisting without storing tokens in a database.

## Features

- JWT-based authentication with access and refresh tokens
- Token blacklisting using Redis for efficient token revocation
- Role-based access control
- Secure password storage with Argon2id
- Token refresh functionality

## Requirements

- Go 1.23.4 or higher
- Redis server (for token blacklisting)

## Setup and Running

1. Clone the repository
2. Install dependencies: `go mod download`
3. Start Redis server
4. Run the application: `go run cmd/server/main.go`

## API Endpoints

- POST /api/auth/login - Login and get tokens
- POST /api/auth/refresh - Refresh access token
- POST /api/auth/logout - Logout (revoke token)
- GET /api/protected - Protected resource (requires authentication)
- GET /api/admin/dashboard - Admin-only resource