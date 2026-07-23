-- +goose Up
CREATE TABLE IF NOT EXISTS journal_users (
    id              BIGSERIAL PRIMARY KEY,
    email           VARCHAR(255) NOT NULL UNIQUE,
    password_hash   VARCHAR(255) NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS active_sessions (
    token           VARCHAR(255) PRIMARY KEY,
    user_id         BIGINT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS journals (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL,
    content         TEXT NOT NULL,
    decision_score  DOUBLE PRECISION NOT NULL DEFAULT 50.0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_configs (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL UNIQUE,
    theme           VARCHAR(50) NOT NULL DEFAULT 'dark',
    notifications   BOOLEAN NOT NULL DEFAULT true
);

CREATE TABLE IF NOT EXISTS llm_metrics (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL,
    latency_ms      BIGINT NOT NULL DEFAULT 0,
    token_count     INT NOT NULL DEFAULT 0,
    error_log       TEXT DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_journal_users_email ON journal_users(email);
CREATE INDEX IF NOT EXISTS idx_journals_user_id ON journals(user_id);
CREATE INDEX IF NOT EXISTS idx_llm_metrics_user_id ON llm_metrics(user_id);
CREATE INDEX IF NOT EXISTS idx_active_sessions_user_id ON active_sessions(user_id);

-- +goose Down
DROP TABLE IF EXISTS llm_metrics;
DROP TABLE IF EXISTS user_configs;
DROP TABLE IF EXISTS journals;
DROP TABLE IF EXISTS active_sessions;
DROP TABLE IF EXISTS journal_users;
