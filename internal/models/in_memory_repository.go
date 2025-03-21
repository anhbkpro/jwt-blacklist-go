package models

import (
	"context"
	"errors"
	"sync"
)

// InMemoryUserRepository implements UserRepository interface with an in-memory store
type InMemoryUserRepository struct {
	Users map[string]*User
	mu    sync.RWMutex // for thread-safety
}

// GetByUsername retrieves a user by username from the in-memory store
func (r *InMemoryUserRepository) GetByUsername(ctx context.Context, username string) (*User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, exists := r.Users[username]
	if !exists {
		return nil, nil // User not found
	}

	// Return a copy to prevent modification of the stored user
	userCopy := *user
	return &userCopy, nil
}

// Create adds a new user to the in-memory store
func (r *InMemoryUserRepository) Create(ctx context.Context, user *User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if username already exists
	if _, exists := r.Users[user.Username]; exists {
		return errors.New("username already exists")
	}

	// Find the next available ID
	maxID := 0
	for _, existingUser := range r.Users {
		if existingUser.ID > maxID {
			maxID = existingUser.ID
		}
	}

	// Assign a new ID if not provided
	if user.ID <= 0 {
		user.ID = maxID + 1
	}

	// Store a copy of the user
	userCopy := *user
	r.Users[user.Username] = &userCopy

	return nil
}

// Update modifies an existing user in the in-memory store
func (r *InMemoryUserRepository) Update(ctx context.Context, user *User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Find the user by username
	var username string
	for uname, existingUser := range r.Users {
		if existingUser.ID == user.ID {
			username = uname
			break
		}
	}

	if username == "" {
		return errors.New("user not found")
	}

	// If username is changing, make sure the new one doesn't exist
	if username != user.Username {
		if _, exists := r.Users[user.Username]; exists {
			return errors.New("new username already exists")
		}
		// Remove the old username entry
		delete(r.Users, username)
	}

	// Store a copy of the updated user
	userCopy := *user
	r.Users[user.Username] = &userCopy

	return nil
}

// Delete removes a user from the in-memory store
func (r *InMemoryUserRepository) Delete(ctx context.Context, id int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Find the user by ID
	var username string
	for uname, user := range r.Users {
		if user.ID == id {
			username = uname
			break
		}
	}

	if username == "" {
		return errors.New("user not found")
	}

	// Delete the user
	delete(r.Users, username)

	return nil
}
