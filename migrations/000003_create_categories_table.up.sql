CREATE TABLE categories (
                            id      BIGSERIAL PRIMARY KEY,
                            user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                            name    TEXT NOT NULL,
                            UNIQUE (user_id, name)
);