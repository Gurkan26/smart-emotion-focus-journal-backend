-- +goose Up
CREATE TABLE IF NOT EXISTS admin_users (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL UNIQUE REFERENCES journal_users(id) ON DELETE CASCADE,
    admin_level     VARCHAR(50) NOT NULL DEFAULT 'superadmin',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS llm_configs (
    id              BIGSERIAL PRIMARY KEY,
    system_prompt   TEXT NOT NULL DEFAULT '',
    max_tokens      INT NOT NULL DEFAULT 2048,
    temperature     DOUBLE PRECISION NOT NULL DEFAULT 0.2,
    top_p           DOUBLE PRECISION NOT NULL DEFAULT 0.9,
    active_adapter  VARCHAR(255) DEFAULT 'gemma-default-lora',
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS peft_adapters (
    id              BIGSERIAL PRIMARY KEY,
    name            VARCHAR(255) NOT NULL UNIQUE,
    description     TEXT DEFAULT '',
    file_path       VARCHAR(512) NOT NULL,
    is_active       BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed default initial LLM config if empty
INSERT INTO llm_configs (system_prompt, max_tokens, temperature, top_p, active_adapter)
SELECT 'You are an expert AI prompt optimizer and cognitive load analyst.', 2048, 0.2, 0.9, 'gemma-default-lora'
WHERE NOT EXISTS (SELECT 1 FROM llm_configs);

-- Seed sample PEFT adapter list
INSERT INTO peft_adapters (name, description, file_path, is_active)
VALUES 
  ('gemma-default-lora', 'Default fine-tuned LoRA adapter for general prompt optimization', '/adapters/gemma-default-lora.safetensors', true),
  ('code-optimizer-v2', 'LoRA adapter fine-tuned on 50k software engineering specs', '/adapters/code-optimizer-v2.safetensors', false),
  ('token-compressor-lora', 'Ultra-low parameter adapter for prompt compression', '/adapters/token-compressor-lora.safetensors', false)
ON CONFLICT (name) DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS peft_adapters;
DROP TABLE IF EXISTS llm_configs;
DROP TABLE IF EXISTS admin_users;
