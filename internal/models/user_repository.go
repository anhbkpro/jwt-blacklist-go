package models

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// PostgresUserRepository implements UserRepository interface for PostgreSQL
type PostgresUserRepository struct {
	db *sql.DB
}

// NewPostgresUserRepository creates a new PostgreSQL user repository
func NewPostgresUserRepository(db *sql.DB) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

// GetByUsername retrieves a user by username
func (r *PostgresUserRepository) GetByUsername(ctx context.Context, username string) (*User, error) {
	query := `
		SELECT id, username, email, password, role
		FROM users
		WHERE username = $1
	`

	// Create a context with timeout
	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Execute the query
	row := r.db.QueryRowContext(queryCtx, query, username)

	// Parse the result
	user := &User{}
	err := row.Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.Role)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // User not found
		}
		return nil, err // Database error
	}

	return user, nil
}

// Create adds a new user to the database
func (r *PostgresUserRepository) Create(ctx context.Context, user *User) error {
	query := `
		INSERT INTO users (username, email, password, role)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`

	// Create a context with timeout
	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Execute the query
	err := r.db.QueryRowContext(
		queryCtx,
		query,
		user.Username,
		user.Email,
		user.Password,
		user.Role,
	).Scan(&user.ID)

	return err
}

// Update modifies an existing user in the database
func (r *PostgresUserRepository) Update(ctx context.Context, user *User) error {
	query := `
		UPDATE users
		SET username = $1, email = $2, password = $3, role = $4, updated_at = CURRENT_TIMESTAMP
		WHERE id = $5
	`

	// Create a context with timeout
	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Execute the query
	_, err := r.db.ExecContext(
		queryCtx,
		query,
		user.Username,
		user.Email,
		user.Password,
		user.Role,
		user.ID,
	)

	return err
}

// Delete removes a user from the database
func (r *PostgresUserRepository) Delete(ctx context.Context, id int) error {
	query := "DELETE FROM users WHERE id = $1"

	// Create a context with timeout
	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Execute the query
	_, err := r.db.ExecContext(queryCtx, query, id)

	return err
}
