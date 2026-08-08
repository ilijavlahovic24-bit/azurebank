package auth

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresStore struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) Store {
	return &postgresStore{db: db}
}

func (s *postgresStore) CreateUser(ctx context.Context, email, passwordHash string) (*User, error) {
	u := &User{}
	err := s.db.QueryRow(ctx,
		`INSERT INTO users (email, password_hash)
		 VALUES ($1, $2)
		 RETURNING id, email, password_hash, created_at`,
		email, passwordHash,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}

func (s *postgresStore) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	u := &User{}
	err := s.db.QueryRow(ctx,
		`SELECT id, email, password_hash, created_at
		 FROM users WHERE email = $1`,
		email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return u, nil
}
