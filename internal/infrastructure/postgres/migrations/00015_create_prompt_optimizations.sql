-- +goose Up
CREATE TABLE IF NOT EXISTS prompt_optimizations (
    id                  BIGSERIAL PRIMARY KEY,
    user_id             BIGINT NOT NULL,
    original_prompt     TEXT NOT NULL,
    optimized_prompt    TEXT NOT NULL,
    template            VARCHAR(50) NOT NULL DEFAULT 'accurate',
    custom_instruction  TEXT DEFAULT '',
    original_tokens     INT NOT NULL DEFAULT 0,
    optimized_tokens    INT NOT NULL DEFAULT 0,
    latency_ms          BIGINT NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_prompt_opt_user_id ON prompt_optimizations(user_id);
CREATE INDEX IF NOT EXISTS idx_prompt_opt_created_at ON prompt_optimizations(created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS prompt_optimizations;
