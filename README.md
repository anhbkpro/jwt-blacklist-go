# JWT Authentication with Token Blacklisting

A Go project demonstrating JWT authentication with token blacklisting without storing tokens in a database.

## Features

- JWT-based authentication with access and refresh tokens
- Token blacklisting using Redis for efficient token revocation
- Role-based access control
- Secure password storage with Argon2id
- Token refresh functionality
- Support for multi-device access and per-device logout

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

## Key Concepts

### Token Blacklisting

Our token blacklisting system uses Redis with automatic expiration (TTL) to maintain a list of invalidated tokens:

* **Tokens not in the blacklist**: allowed access
* **Tokens in the blacklist**: denied access
* **Tokens whose blacklist entry expired**: allowed access (because by then the token itself has expired too)

### Multi-Device Support

The system properly handles multi-device scenarios:

* Each device receives a unique access token with a unique JTI (JWT ID)
* When a user logs out from one device, only that specific token is blacklisted
* Sessions on other devices remain active and unaffected

For example:
* User X can access the app from iPhone and iPad simultaneously
* When User X logs out from iPad, they can still access from iPhone
* Each device manages its own session independently

## Testing with Redis

Monitor blacklisted tokens in Redis CLI:

```
# List all blacklisted tokens
KEYS blacklist:*

# Check if a specific token is blacklisted
EXISTS blacklist:<token-id>

# Check how long until a blacklisted token expires
TTL blacklist:<token-id>
```

## Architecture

The system uses a stateless JWT authentication mechanism where:

1. Tokens are validated by signature, not by database lookup
2. Only revoked tokens are tracked (in Redis)
3. Token expiration is enforced by both the JWT itself and the Redis TTL
4. No background cleanup jobs are needed for maintenance

See the [Token Blacklisting Documentation](./docs/token-blacklisting.md) for more details on the implementation.