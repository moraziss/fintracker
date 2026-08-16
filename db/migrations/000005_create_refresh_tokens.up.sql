CREATE TABLE refresh_tokens (
                                id         BIGSERIAL PRIMARY KEY,
                                user_id    BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                                token_hash TEXT NOT NULL UNIQUE,
                                expires_at TIMESTAMPTZ NOT NULL,
                                created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                                revoked_at TIMESTAMPTZ
);