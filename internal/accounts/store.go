package accounts

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrAccountNotFound = errors.New("account not found")

type postgresStore struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) Store {
	return &postgresStore{db: db}
}

func (s *postgresStore) CreateAccount(ctx context.Context, userID int64, accountType string) (*Account, error) {
	a := &Account{}
	err := s.db.QueryRow(ctx,
		`INSERT INTO accounts (user_id, type)
		 VALUES ($1, $2)
		 RETURNING id, user_id, type, balance, created_at`,
		userID, accountType,
	).Scan(&a.ID, &a.UserID, &a.Type, &a.Balance, &a.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create account: %w", err)
	}
	return a, nil
}

func (s *postgresStore) GetAccountsByUserID(ctx context.Context, userID int64) ([]Account, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, user_id, type, balance, created_at
		 FROM accounts
		 WHERE user_id = $1
		 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("query accounts: %w", err)
	}
	defer rows.Close()

	accounts := []Account{} // ne nil — da JSON vrati [] umesto null
	for rows.Next() {
		var a Account
		if err := rows.Scan(&a.ID, &a.UserID, &a.Type, &a.Balance, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan account: %w", err)
		}
		accounts = append(accounts, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return accounts, nil
}

func (s *postgresStore) GetAccountByID(ctx context.Context, id, userID int64) (*Account, error) {
	a := &Account{}
	err := s.db.QueryRow(ctx,
		`SELECT id, user_id, type, balance, created_at
		 FROM accounts
		 WHERE id = $1 AND user_id = $2`,
		id, userID,
	).Scan(&a.ID, &a.UserID, &a.Type, &a.Balance, &a.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAccountNotFound
		}
		return nil, fmt.Errorf("get account: %w", err)
	}
	return a, nil
}
