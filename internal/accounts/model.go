package accounts

import (
	"context"
	"time"
)

type Account struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Type      string    `json:"type"`
	Balance   int64     `json:"balance"`
	CreatedAt time.Time `json:"created_at"`
}

type Store interface {
	CreateAccount(ctx context.Context, userID int64, accountType string) (*Account, error)
	GetAccountsByUserID(ctx context.Context, userID int64) ([]Account, error)
	GetAccountByID(ctx context.Context, id, userID int64) (*Account, error)
}
