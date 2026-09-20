CREATE TABLE transactions (
    id             BIGSERIAL PRIMARY KEY,
    account_id     BIGINT NOT NULL REFERENCES accounts(id),
    type           TEXT NOT NULL CHECK (type IN ('deposit', 'withdrawal', 'transfer')),
    amount         BIGINT NOT NULL CHECK (amount > 0),
    to_account_id  BIGINT REFERENCES accounts(id),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_transactions_account_id     ON transactions(account_id);
CREATE INDEX idx_transactions_to_account_id  ON transactions(to_account_id);
CREATE INDEX idx_transactions_created_at     ON transactions(created_at DESC);