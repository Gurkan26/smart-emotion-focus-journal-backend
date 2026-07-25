-- +goose Up
ALTER TABLE journal_users ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ DEFAULT NULL;
CREATE INDEX IF NOT EXISTS idx_journal_users_deleted_at ON journal_users(deleted_at);

-- +goose Down
ALTER TABLE journal_users DROP COLUMN IF EXISTS deleted_at;
