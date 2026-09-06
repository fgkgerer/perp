package db

import (
	"fmt"

	"gorm.io/gorm"
)

// MigrateAccountTables 确保 users / deposits / ledger_balances / orders / trades 存在。
// Supabase 上若已有旧表结构，会先 ADD COLUMN IF NOT EXISTS 再建索引。
func MigrateAccountTables(pg *gorm.DB) error {
	if pg == nil {
		return fmt.Errorf("postgres nil")
	}
	batches := [][]string{
		userStmts(),
		depositStmts(),
		ledgerStmts(),
		orderStmts(),
		tradeStmts(),
		engineEventStmts(),
		fundingRateStmts(),
		marketQuoteStmts(),
		spotKlineStmts(),
	}
	for _, batch := range batches {
		for _, s := range batch {
			if err := pg.Exec(s).Error; err != nil {
				return fmt.Errorf("%w\nSQL: %s", err, s)
			}
		}
	}
	if err := pg.Exec(`ALTER TABLE public.market_quotes REPLICA IDENTITY FULL`).Error; err != nil {
		// 非 Supabase 或权限不足时可忽略
		_ = err
	}
	return nil
}

func userStmts() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS public.users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid()
)`,
		`ALTER TABLE public.users ADD COLUMN IF NOT EXISTS wallet_address text NOT NULL DEFAULT ''`,
		`ALTER TABLE public.users ADD COLUMN IF NOT EXISTS created_at timestamptz NOT NULL DEFAULT now()`,
		`ALTER TABLE public.users ADD COLUMN IF NOT EXISTS updated_at timestamptz NOT NULL DEFAULT now()`,
		`ALTER TABLE public.users ADD COLUMN IF NOT EXISTS last_login_at timestamptz`,
		`CREATE UNIQUE INDEX IF NOT EXISTS users_wallet_address_lower_idx ON public.users (lower(wallet_address))`,
	}
}

func depositStmts() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS public.deposits (
    id BIGSERIAL PRIMARY KEY,
    tx_hash VARCHAR(66) NOT NULL,
    log_index BIGINT NOT NULL,
    trader VARCHAR(42) NOT NULL,
    primary_amount TEXT NOT NULL DEFAULT '0',
    secondary_amount TEXT NOT NULL DEFAULT '0',
    block_number BIGINT NOT NULL DEFAULT 0,
    create_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_deposits_tx_log UNIQUE (tx_hash, log_index)
)`,
		`CREATE INDEX IF NOT EXISTS idx_deposits_trader_time ON public.deposits (trader, create_time DESC)`,
	}
}

func ledgerStmts() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS public.ledger_balances (
    trader VARCHAR(42) PRIMARY KEY,
    primary_balance TEXT NOT NULL DEFAULT '0',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)`,
	}
}

func orderStmts() []string {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS public.orders (
    id BIGSERIAL PRIMARY KEY
)`,
	}
	// 旧 Supabase 表可能只有 id，逐列补齐后再建索引
	cols := []string{
		`order_id VARCHAR(66) NOT NULL DEFAULT ''`,
		`chain_id BIGINT NOT NULL DEFAULT 11155111`,
		`perp VARCHAR(42) NOT NULL DEFAULT ''`,
		`perp_address VARCHAR(42) NOT NULL DEFAULT ''`,
		`signer VARCHAR(42) NOT NULL DEFAULT ''`,
		`paper_amount TEXT NOT NULL DEFAULT '0'`,
		`credit_amount TEXT NOT NULL DEFAULT '0'`,
		`maker_fee_rate TEXT NOT NULL DEFAULT '0'`,
		`taker_fee_rate TEXT NOT NULL DEFAULT '0'`,
		`expiration BIGINT NOT NULL DEFAULT 0`,
		`nonce BIGINT NOT NULL DEFAULT 0`,
		`signature TEXT NOT NULL DEFAULT ''`,
		`status INT NOT NULL DEFAULT 0`,
		`filled_amount TEXT NOT NULL DEFAULT '0'`,
		`create_time TIMESTAMPTZ NOT NULL DEFAULT NOW()`,
		`update_time TIMESTAMPTZ NOT NULL DEFAULT NOW()`,
	}
	for _, col := range cols {
		stmts = append(stmts, fmt.Sprintf(
			`ALTER TABLE public.orders ADD COLUMN IF NOT EXISTS %s`, col))
	}
	// 删除旧 schema 遗留的外键约束（orders_fk_market 引用 perp_markets，后端不维护该表）
	dropFKStmts := []string{
		`DO $$ BEGIN
			IF EXISTS (
				SELECT 1 FROM information_schema.table_constraints
				WHERE table_schema='public' AND table_name='orders'
				  AND constraint_name='orders_fk_market'
			) THEN
				ALTER TABLE public.orders DROP CONSTRAINT orders_fk_market;
			END IF;
		END $$`,
		// 同样处理可能存在的其他市场外键
		`DO $$ BEGIN
			IF EXISTS (
				SELECT 1 FROM information_schema.table_constraints
				WHERE table_schema='public' AND table_name='orders'
				  AND constraint_type='FOREIGN KEY'
				  AND constraint_name LIKE '%market%'
			) THEN
				EXECUTE (
					SELECT 'ALTER TABLE public.orders DROP CONSTRAINT ' || constraint_name
					FROM information_schema.table_constraints
					WHERE table_schema='public' AND table_name='orders'
					  AND constraint_type='FOREIGN KEY'
					  AND constraint_name LIKE '%market%'
					LIMIT 1
				);
			END IF;
		END $$`,
	}
	stmts = append(stmts, dropFKStmts...)
	stmts = append(stmts,
		`ALTER TABLE public.orders ALTER COLUMN chain_id SET DEFAULT 11155111`,
		`UPDATE public.orders SET perp = perp_address WHERE perp = '' AND perp_address <> ''`,
		`UPDATE public.orders SET perp_address = perp WHERE perp_address = '' AND perp <> ''`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_orders_order_id ON public.orders (order_id) WHERE order_id <> ''`,
		`CREATE INDEX IF NOT EXISTS idx_orders_signer_time ON public.orders (signer, create_time DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_orders_perp_status ON public.orders (perp, status)`,
	)
	return stmts
}

func tradeStmts() []string {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS public.trades (
    id BIGSERIAL PRIMARY KEY
)`,
	}
	cols := []string{
		`trade_id VARCHAR(64) NOT NULL DEFAULT ''`,
		`perp VARCHAR(42) NOT NULL DEFAULT ''`,
		`taker_order_id VARCHAR(66) NOT NULL DEFAULT ''`,
		`maker_order_id VARCHAR(66) NOT NULL DEFAULT ''`,
		`taker VARCHAR(42) NOT NULL DEFAULT ''`,
		`maker VARCHAR(42) NOT NULL DEFAULT ''`,
		`paper_amount TEXT NOT NULL DEFAULT '0'`,
		`price TEXT NOT NULL DEFAULT '0'`,
		`taker_fee TEXT NOT NULL DEFAULT '0'`,
		`maker_fee TEXT NOT NULL DEFAULT '0'`,
		`tx_hash VARCHAR(66) NOT NULL DEFAULT ''`,
		`block_number BIGINT NOT NULL DEFAULT 0`,
		`create_time TIMESTAMPTZ NOT NULL DEFAULT NOW()`,
	}
	for _, col := range cols {
		stmts = append(stmts, fmt.Sprintf(
			`ALTER TABLE public.trades ADD COLUMN IF NOT EXISTS %s`, col))
	}
	stmts = append(stmts,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_trades_trade_id ON public.trades (trade_id) WHERE trade_id <> ''`,
		`CREATE INDEX IF NOT EXISTS idx_trades_perp_time ON public.trades (perp, create_time DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_trades_taker ON public.trades (taker, create_time DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_trades_maker ON public.trades (maker, create_time DESC)`,
	)
	return stmts
}

func engineEventStmts() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS public.engine_events (
    id BIGSERIAL PRIMARY KEY,
    event_type VARCHAR(64) NOT NULL,
    order_id VARCHAR(66) NOT NULL DEFAULT '',
    match_id VARCHAR(80) NOT NULL DEFAULT '',
    perp VARCHAR(42) NOT NULL DEFAULT '',
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    node_id VARCHAR(128) NOT NULL DEFAULT '',
    create_time TIMESTAMPTZ NOT NULL DEFAULT NOW()
)`,
		`CREATE INDEX IF NOT EXISTS idx_engine_events_type_id ON public.engine_events (event_type, id)`,
		`CREATE INDEX IF NOT EXISTS idx_engine_events_order_id ON public.engine_events (order_id)`,
		`CREATE INDEX IF NOT EXISTS idx_engine_events_match_id ON public.engine_events (match_id) WHERE match_id <> ''`,
		`CREATE INDEX IF NOT EXISTS idx_engine_events_perp_id ON public.engine_events (perp, id)`,
	}
}

func fundingRateStmts() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS public.funding_rates (
    id BIGSERIAL PRIMARY KEY,
    perp VARCHAR(42) NOT NULL,
    rate TEXT NOT NULL DEFAULT '0',
    mark_price TEXT NOT NULL DEFAULT '0',
    index_price TEXT NOT NULL DEFAULT '0',
    settle_time TIMESTAMPTZ NOT NULL DEFAULT NOW()
)`,
		`CREATE INDEX IF NOT EXISTS idx_funding_rates_perp_time ON public.funding_rates (perp, settle_time DESC)`,
	}
}

func spotKlineStmts() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS public.spot_klines (
    id BIGSERIAL PRIMARY KEY,
    perp VARCHAR(42) NOT NULL,
    interval_type VARCHAR(8) NOT NULL,
    open_time TIMESTAMPTZ NOT NULL,
    open_price TEXT NOT NULL DEFAULT '0',
    high_price TEXT NOT NULL DEFAULT '0',
    low_price TEXT NOT NULL DEFAULT '0',
    close_price TEXT NOT NULL DEFAULT '0',
    volume TEXT NOT NULL DEFAULT '0',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_spot_klines_perp_interval_time UNIQUE (perp, interval_type, open_time)
)`,
		`CREATE INDEX IF NOT EXISTS idx_spot_klines_perp_interval_time ON public.spot_klines (perp, interval_type, open_time DESC)`,
	}
}

func marketQuoteStmts() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS public.market_quotes (
    perp VARCHAR(42) PRIMARY KEY,
    market_name TEXT NOT NULL DEFAULT '',
    product_id TEXT NOT NULL DEFAULT '',
    price_usd TEXT NOT NULL DEFAULT '0',
    open_24h TEXT NOT NULL DEFAULT '0',
    volume_24h TEXT NOT NULL DEFAULT '0',
    low_24h TEXT NOT NULL DEFAULT '0',
    high_24h TEXT NOT NULL DEFAULT '0',
    price_change_24h TEXT NOT NULL DEFAULT '0',
    price_change_percent_24h TEXT NOT NULL DEFAULT '0',
    source TEXT NOT NULL DEFAULT 'coinbase',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)`,
		`CREATE INDEX IF NOT EXISTS idx_market_quotes_updated_at ON public.market_quotes (updated_at DESC)`,
	}
}
