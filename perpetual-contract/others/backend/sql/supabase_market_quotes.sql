-- MetaNode 行情表：后端 Coinbase WS 写入，前端 Supabase Realtime 订阅
-- 在 Supabase Dashboard → SQL Editor 中整段执行

-- 1. 表结构（每个 perp 合约一行，UPSERT 更新）
CREATE TABLE IF NOT EXISTS public.market_quotes (
    perp VARCHAR(42) PRIMARY KEY,
    market_name TEXT NOT NULL,
    product_id TEXT NOT NULL,
    price_usd TEXT NOT NULL DEFAULT '0',
    open_24h TEXT NOT NULL DEFAULT '0',
    volume_24h TEXT NOT NULL DEFAULT '0',
    low_24h TEXT NOT NULL DEFAULT '0',
    high_24h TEXT NOT NULL DEFAULT '0',
    price_change_24h TEXT NOT NULL DEFAULT '0',
    price_change_percent_24h TEXT NOT NULL DEFAULT '0',
    source TEXT NOT NULL DEFAULT 'coinbase',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_market_quotes_updated_at
    ON public.market_quotes (updated_at DESC);

-- 2. UPDATE 事件需要完整行（UPSERT 走 UPDATE）
ALTER TABLE public.market_quotes REPLICA IDENTITY FULL;

-- 3. 开启 Realtime（postgres_changes）
-- 若报错 "relation already member of publication" 可忽略
ALTER PUBLICATION supabase_realtime ADD TABLE public.market_quotes;

-- 3. RLS：仅允许匿名/登录用户读，禁止前端写
ALTER TABLE public.market_quotes ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS market_quotes_select_anon ON public.market_quotes;
CREATE POLICY market_quotes_select_anon
    ON public.market_quotes
    FOR SELECT
    TO anon, authenticated
    USING (true);

-- 后端使用 postgres 直连（service role / database URL）写入，不走 RLS

-- 4. 验证
-- SELECT * FROM public.market_quotes;
