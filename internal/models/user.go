package models

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"log"

	"golang.org/x/crypto/argon2"
)

// User represents user data in the system
type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"-"` // Hashed password, never returned in JSON
	Role     string `json:"role"`
}

// PasswordParams stores parameters used for password hashing
type PasswordParams struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

var defaultParams = &PasswordParams{
	Memory:      64 * 1024,
	Iterations:  3,
	Parallelism: 2,
	SaltLength:  16,
	KeyLength:   32,
}

// HashPassword creates a new password hash using Argon2id
func HashPassword(password string) (string, error) {
	p := defaultParams

	// Generate a random salt
	salt := make([]byte, p.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	// Hash the password
	hash := argon2.IDKey([]byte(password), salt, p.Iterations, p.Memory, p.Parallelism, p.KeyLength)

	// Encode as base64
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	// Format: $argon2id$v=19$m=memory,t=iterations,p=parallelism$salt$hash
	encodedHash := fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", p.Memory, p.Iterations, p.Parallelism, b64Salt, b64Hash)

	return encodedHash, nil
}

// VerifyPassword checks if the provided password matches the stored hash
func VerifyPassword(password, encodedHash string) (bool, error) {
	// Extract parameters, salt and hash from the encoded hash
	params, salt, hash, err := decodeHash(encodedHash)
	if err != nil {
		return false, err
	}
	log.Println(params, salt, hash)

	// Hash the password with the same parameters and salt
	newHash := argon2.IDKey([]byte(password), salt, params.Iterations, params.Memory, params.Parallelism, params.KeyLength)
	log.Println(newHash)

	// Check if the hashes match using a constant-time comparison
	return subtle.ConstantTimeCompare(hash, newHash) == 1, nil
}

// Helper function to decode a password hash
func decodeHash(encodedHash string) (*PasswordParams, []byte, []byte, error) {
	var params PasswordParams
	var salt, hash []byte

	// Format: $argon2id$v=19$m=memory,t=iterations,p=parallelism$salt$hash
	_, err := fmt.Sscanf(encodedHash, "$argon2id$v=19$m=%d,t=%d,p=%d$", &params.Memory, &params.Iterations, &params.Parallelism)
	if err != nil {
		return nil, nil, nil, errors.New("invalid hash format")
	}

	// Split the string
	parts := splitHashString(encodedHash)
	if len(parts) != 6 {
		return nil, nil, nil, errors.New("invalid hash format")
	}

	// Decode salt
	salt, err = base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, nil, nil, errors.New("invalid salt encoding")
	}
	params.SaltLength = uint32(len(salt))

	// Decode hash
	hash, err = base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return nil, nil, nil, errors.New("invalid hash encoding")
	}
	params.KeyLength = uint32(len(hash))

	return &params, salt, hash, nil
}

// Helper function to split the hash string by $ delimiters
func splitHashString(encodedHash string) []string {
	var result []string
	var part string

	for i := 0; i < len(encodedHash); i++ {
		if encodedHash[i] == '$' {
			result = append(result, part)
			part = ""
		} else {
			part += string(encodedHash[i])
		}
	}

	if part != "" {
		result = append(result, part)
	}

	return result
}

// In-memory user store for demonstration
var UserStore = map[string]*User{
	"admin": {
		ID:       1,
		Username: "admin",
		Email:    "admin@example.com",
		// Default password: "admin123"
		Password: "$argon2id$v=19$m=65536,t=3,p=2$mwTVNvIy4EBaphLMv6Iozg$HrAc8MQ/g1HX6eryWcFc75h7vknOqADznwS6zA04REw",
		Role:     "admin",
	},
	"user": {
		ID:       2,
		Username: "user",
		Email:    "user@example.com",
		// Default password: "user123"
		Password: "$argon2id$v=19$m=65536,t=3,p=2$P00D1MRXhY+tSrYMCDe0rg$JkUThHcvsIxD1RW+5zGqCfvzbtK2+RQ5iV6jyH/OcjI",
		Role:     "user",
	},
}

// GetUserByUsername retrieves a user by username
func GetUserByUsername(username string) (*User, bool) {
	user, exists := UserStore[username]
	return user, exists
}
