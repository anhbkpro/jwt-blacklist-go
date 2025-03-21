# Password Hash Utilities

This package provides utilities for generating and verifying Argon2id password hashes for use with the JWT authentication system.

## Password Hash Command Line Tool

A command-line tool is provided for generating password hashes. This is particularly useful for:
- Creating new user accounts
- Updating user passwords
- Testing password verification

### Building the Tool

```bash
# From the project root
go build -o bin/hash cmd/hash/main.go
```

### Using the Tool

#### 1. Hashing Multiple Passwords

```bash
# Using short flag
./bin/hash -p "user123,admin123"

# Using long flag
./bin/hash --passwords "user123,admin123"
```

#### 2. Interactive Mode

```bash
# Using short flag
./bin/hash -i

# Using long flag
./bin/hash --interactive
```

In interactive mode, you'll be prompted to enter passwords one per line. Press Ctrl+D (Unix) or Ctrl+Z (Windows) followed by Enter when done.

### Example Output

```
=============================================
Password hash for 'user123':
$argon2id$v=19$m=65536,t=3,p=2$mwTVNvIy4EBaphLMv6Iozg$HrAc8MQ/g1HX6eryWcFc75h7vknOqADznwS6zA04REw
---------------------------------------------
Password hash for 'admin123':
$argon2id$v=19$m=65536,t=3,p=2$P00D1MRXhY+tSrYMCDe0rg$JkUThHcvsIxD1RW+5zGqCfvzbtK2+RQ5iV6jyH/OcjI
---------------------------------------------
=============================================
```

## Programmatic Usage

The password utilities can also be used programmatically in your Go code:

```go
import "github.com/anhbkpro/jwt-blacklist-go/internal/utils"

// Generate and print a single password hash
utils.PrintPasswordHash("mypassword")

// Generate and print multiple password hashes
utils.PrintMultiplePasswordHashes("password1", "password2", "password3")

// Generate a password hash and get the result as a string
hash := utils.GeneratePasswordHash("mypassword")

// Generate multiple password hashes and get a map of password to hash
hashMap := utils.GeneratePasswordHashes("password1", "password2")
```

## Security Considerations

- The generated hashes use Argon2id with secure parameters
- Each hash includes a unique random salt
- The same password will generate different hashes each time due to the random salt
- Verification is done in constant time to prevent timing attacks
