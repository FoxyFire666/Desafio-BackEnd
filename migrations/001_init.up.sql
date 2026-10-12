CREATE TABLE IF NOT EXISTS wallets (
    id UUID PRIMARY KEY,
    player_id TEXT NOT NULL,
    currency TEXT NOT NULL,
    balance_minor BIGINT NOT NULL DEFAULT 0,
    version BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (player_id, currency)
);

CREATE TABLE IF NOT EXISTS wager_transactions (
    id UUID PRIMARY KEY,
    provider_id TEXT NOT NULL,
    external_transaction_id TEXT NOT NULL,
    idempotency_key TEXT NOT NULL,
    player_id TEXT NOT NULL,
    wallet_id UUID NOT NULL REFERENCES wallets(id),
    round_id TEXT NOT NULL,
    game_id TEXT NOT NULL,
    kind TEXT NOT NULL,
    amount_minor BIGINT NOT NULL,
    currency TEXT NOT NULL,
    reference_external_transaction_id TEXT NULL,
    status TEXT NOT NULL DEFAULT 'PENDING',
    failure_code TEXT NULL,
    result_amount_minor BIGINT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_internal_transaction_id TEXT NULL,
    business_hash TEXT NULL,
    UNIQUE (provider_id, external_transaction_id),
    UNIQUE (idempotency_key)
);

CREATE TABLE IF NOT EXISTS ledger (
    id UUID PRIMARY KEY,
    wallet_id UUID NOT NULL REFERENCES wallets(id),
    entry_type TEXT NOT NULL,
    amount_minor BIGINT NOT NULL,
    currency TEXT NOT NULL,
    balance_minor BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS outbox (
    id UUID PRIMARY KEY,
    aggregate_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    published BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
