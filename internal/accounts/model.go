package accounts

import "time"

type Account struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Type      string    `json:"type"`    // "checking" | "savings"
	Balance   int64     `json:"balance"` // u dinarima/centima, nikad float
	CreatedAt time.Time `json:"created_at"`
}
