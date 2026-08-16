package accounts

import "context"

type Store interface {
	CreateAccount(ctx context.Context, userID int64, accountType string) (*Account, error)
	GetAccountsByUserID(ctx context.Context, userID int64) ([]Account, error)
	GetAccountByID(ctx context.Context, id, userID int64) (*Account, error)
}
