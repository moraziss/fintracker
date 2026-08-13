CREATE TABLE transactions (
                              id          BIGSERIAL PRIMARY KEY,
                              account_id  BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
                              category_id BIGINT REFERENCES categories(id) ON DELETE SET NULL,
                              type        TEXT NOT NULL CHECK (type IN ('income', 'expense')),
                              amount      NUMERIC(14,2) NOT NULL CHECK (amount > 0),
                              description TEXT,
                              occurred_at DATE NOT NULL DEFAULT CURRENT_DATE,
                              created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
                              deleted_at  TIMESTAMPTZ
);