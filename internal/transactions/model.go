package transactions

import (
	"context"
	"errors"
	"time"
)

type Transaction struct {
	ID          int64     `json:"id"`
	AccountID   int64     `json:"account_id"`
	Type        string    `json:"type"`
	Amount      int64     `json:"amount"`
	ToAccountID *int64    `json:"to_account_id"`
	CreatedAt   time.Time `json:"created_at"`
}

var (
	ErrAccountNotFound   = errors.New("account not found")
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrSameAccount       = errors.New("from and to accounts must be different")
)

type Store interface {
	Deposit(ctx context.Context, userID, accountID, amount int64) (*Transaction, error)
	Withdraw(ctx context.Context, userID, accountID, amount int64) (*Transaction, error)
	Transfer(ctx context.Context, userID, fromID, toID, amount int64) (*Transaction, error)
	ListByAccount(ctx context.Context, userID, accountID int64) ([]Transaction, error)
}
