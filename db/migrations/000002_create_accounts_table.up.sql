CREATE TABLE accounts (
                          id         BIGSERIAL PRIMARY KEY,
                          user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                          name       TEXT NOT NULL,
                          currency   CHAR(3) NOT NULL DEFAULT 'KZT',
                          balance    NUMERIC(14,2) NOT NULL DEFAULT 0,
                          created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                          UNIQUE (user_id, name)
);