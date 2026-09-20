package transactions

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresStore struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) Store {
	return &postgresStore{db: db}
}

func (s *postgresStore) Deposit(ctx context.Context, userID, accountID, amount int64) (*Transaction, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(ctx)

	// Lock + ownership check
	var dummy int64
	err = tx.QueryRow(ctx,
		`SELECT id FROM accounts WHERE id = $1 AND user_id = $2 FOR UPDATE`,
		accountID, userID,
	).Scan(&dummy)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAccountNotFound
		}
		return nil, fmt.Errorf("lock account: %w", err)
	}

	if _, err := tx.Exec(ctx,
		`UPDATE accounts SET balance = balance + $1 WHERE id = $2`,
		amount, accountID,
	); err != nil {
		return nil, fmt.Errorf("update balance: %w", err)
	}

	t := &Transaction{}
	err = tx.QueryRow(ctx,
		`INSERT INTO transactions (account_id, type, amount)
		 VALUES ($1, 'deposit', $2)
		 RETURNING id, account_id, type, amount, to_account_id, created_at`,
		accountID, amount,
	).Scan(&t.ID, &t.AccountID, &t.Type, &t.Amount, &t.ToAccountID, &t.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert tx: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return t, nil
}

func (s *postgresStore) Withdraw(ctx context.Context, userID, accountID, amount int64) (*Transaction, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(ctx)

	var balance int64
	err = tx.QueryRow(ctx,
		`SELECT balance FROM accounts WHERE id = $1 AND user_id = $2 FOR UPDATE`,
		accountID, userID,
	).Scan(&balance)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAccountNotFound
		}
		return nil, fmt.Errorf("lock account: %w", err)
	}

	if balance < amount {
		return nil, ErrInsufficientFunds
	}

	if _, err := tx.Exec(ctx,
		`UPDATE accounts SET balance = balance - $1 WHERE id = $2`,
		amount, accountID,
	); err != nil {
		return nil, fmt.Errorf("update balance: %w", err)
	}

	t := &Transaction{}
	err = tx.QueryRow(ctx,
		`INSERT INTO transactions (account_id, type, amount)
		 VALUES ($1, 'withdrawal', $2)
		 RETURNING id, account_id, type, amount, to_account_id, created_at`,
		accountID, amount,
	).Scan(&t.ID, &t.AccountID, &t.Type, &t.Amount, &t.ToAccountID, &t.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert tx: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return t, nil
}

func (s *postgresStore) Transfer(ctx context.Context, userID, fromID, toID, amount int64) (*Transaction, error) {
	if fromID == toID {
		return nil, ErrSameAccount
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(ctx)
	//Lock both accounts in a consistent order (ascending ID) to prevent deadlocks
	// between concurrent transfers A→B and B→A.
	first, second := fromID, toID
	if first > second {
		first, second = second, first
	}

	for _, id := range []int64{first, second} {
		var dummy int64
		if err := tx.QueryRow(ctx,
			`SELECT id FROM accounts WHERE id = $1 FOR UPDATE`, id,
		).Scan(&dummy); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrAccountNotFound
			}
			return nil, fmt.Errorf("lock account %d: %w", id, err)
		}
	}

	// Check ownership of the "from" account
	var fromBalance int64
	err = tx.QueryRow(ctx,
		`SELECT balance FROM accounts WHERE id = $1 AND user_id = $2`,
		fromID, userID,
	).Scan(&fromBalance)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAccountNotFound
		}
		return nil, fmt.Errorf("read from balance: %w", err)
	}

	if fromBalance < amount {
		return nil, ErrInsufficientFunds
	}

	if _, err := tx.Exec(ctx,
		`UPDATE accounts SET balance = balance - $1 WHERE id = $2`,
		amount, fromID,
	); err != nil {
		return nil, fmt.Errorf("deduct: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`UPDATE accounts SET balance = balance + $1 WHERE id = $2`,
		amount, toID,
	); err != nil {
		return nil, fmt.Errorf("credit: %w", err)
	}

	t := &Transaction{}
	err = tx.QueryRow(ctx,
		`INSERT INTO transactions (account_id, type, amount, to_account_id)
		 VALUES ($1, 'transfer', $2, $3)
		 RETURNING id, account_id, type, amount, to_account_id, created_at`,
		fromID, amount, toID,
	).Scan(&t.ID, &t.AccountID, &t.Type, &t.Amount, &t.ToAccountID, &t.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert tx: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return t, nil
}

func (s *postgresStore) ListByAccount(ctx context.Context, userID, accountID int64) ([]Transaction, error) {
	// Provera vlasništva
	var exists bool
	if err := s.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM accounts WHERE id = $1 AND user_id = $2)`,
		accountID, userID,
	).Scan(&exists); err != nil {
		return nil, fmt.Errorf("check ownership: %w", err)
	}
	if !exists {
		return nil, ErrAccountNotFound
	}

	rows, err := s.db.Query(ctx,
		`SELECT id, account_id, type, amount, to_account_id, created_at
		 FROM transactions
		 WHERE account_id = $1 OR to_account_id = $1
		 ORDER BY created_at DESC`,
		accountID,
	)
	if err != nil {
		return nil, fmt.Errorf("query tx: %w", err)
	}
	defer rows.Close()

	out := []Transaction{}
	for rows.Next() {
		var t Transaction
		if err := rows.Scan(&t.ID, &t.AccountID, &t.Type, &t.Amount, &t.ToAccountID, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan tx: %w", err)
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return out, nil
}
