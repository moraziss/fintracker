CREATE INDEX idx_accounts_user_id ON accounts(user_id);

CREATE INDEX idx_transactions_account_occurred
    ON transactions (account_id, occurred_at DESC, id DESC)
    WHERE deleted_at IS NULL;