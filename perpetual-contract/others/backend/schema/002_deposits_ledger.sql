-- MetaNode 链下充值 / 余额（Supabase public schema）
-- 在 Supabase SQL Editor 执行；服务启动时也会 AutoMigrate 对齐。

CREATE TABLE IF NOT EXISTS public.deposits (
    id BIGSERIAL PRIMARY KEY,
    tx_hash VARCHAR(66) NOT NULL,
    log_index BIGINT NOT NULL,
    trader VARCHAR(42) NOT NULL,
    primary_amount TEXT NOT NULL DEFAULT '0',
    secondary_amount TEXT NOT NULL DEFAULT '0',
    block_number BIGINT NOT NULL DEFAULT 0,
    create_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_deposits_tx_log UNIQUE (tx_hash, log_index)
);

CREATE INDEX IF NOT EXISTS idx_deposits_trader_time ON public.deposits (trader, create_time DESC);

CREATE TABLE IF NOT EXISTS public.ledger_balances (
    trader VARCHAR(42) PRIMARY KEY,
    primary_balance TEXT NOT NULL DEFAULT '0',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
