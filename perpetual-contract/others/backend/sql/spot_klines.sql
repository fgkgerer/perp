-- 现货指数 K 线（Supabase / Postgres）
-- 服务启动时也会通过 db.MigrateAccountTables 自动创建

CREATE TABLE IF NOT EXISTS public.spot_klines (
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
);

CREATE INDEX IF NOT EXISTS idx_spot_klines_perp_interval_time
    ON public.spot_klines (perp, interval_type, open_time DESC);
