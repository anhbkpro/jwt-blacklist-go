package utils

import (
	"fmt"
	"log"

	"github.com/anhbkpro/jwt-blacklist-go/internal/models"
)

// GeneratePasswordHash creates a password hash for a given password
// and prints the result with verification
func GeneratePasswordHash(password string) string {
	hash, err := models.HashPassword(password)
	if err != nil {
		log.Fatalf("Error hashing password: %v", err)
	}

	// Verify the hash
	valid, err := models.VerifyPassword(password, hash)
	if err != nil {
		log.Fatalf("Error verifying password: %v", err)
	}

	if !valid {
		log.Fatalf("Verification failed for password hash")
	}

	return hash
}

// GeneratePasswordHashes creates password hashes for multiple passwords
// and returns a map of password to hash
func GeneratePasswordHashes(passwords ...string) map[string]string {
	result := make(map[string]string)

	for _, password := range passwords {
		hash := GeneratePasswordHash(password)
		result[password] = hash
	}

	return result
}

// PrintPasswordHash generates and prints a password hash
func PrintPasswordHash(password string) {
	hash := GeneratePasswordHash(password)
	fmt.Println("=============================================")
	fmt.Printf("Password hash for '%s':\n", password)
	fmt.Println(hash)
	fmt.Println("=============================================")
}

// PrintMultiplePasswordHashes generates and prints hashes for multiple passwords
func PrintMultiplePasswordHashes(passwords ...string) {
	fmt.Println("=============================================")
	for _, password := range passwords {
		hash := GeneratePasswordHash(password)
		fmt.Printf("Password hash for '%s':\n", password)
		fmt.Println(hash)
		fmt.Println("---------------------------------------------")
	}
	fmt.Println("=============================================")
}
