-- MetaNode：钱包签名登录后的应用用户表（public.users）
-- 说明：这与 Supabase Auth 自带的 auth.users 不是同一张表；
--       auth.users 由 Supabase 认证服务维护，勿在本服务中直接 INSERT。
--
-- 在 Supabase Dashboard → SQL Editor 执行一次即可。

CREATE TABLE IF NOT EXISTS public.users (
    id BIGSERIAL PRIMARY KEY,
    wallet_address VARCHAR(42) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_login_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE public.users IS 'MetaNode wallet-login users (JWT subject = wallet_address lowercase)';
