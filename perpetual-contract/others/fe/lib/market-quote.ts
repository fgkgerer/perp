/** Supabase public.market_quotes 行（Coinbase 指数价） */

export type MarketQuoteRow = {
  perp: string;
  market_name: string;
  product_id: string;
  price_usd: string;
  open_24h: string;
  volume_24h: string;
  low_24h: string;
  high_24h: string;
  price_change_24h: string;
  price_change_percent_24h: string;
  source: string;
  updated_at: string;
};

export function quotePriceUsd(raw: string | undefined | null): number {
  if (!raw) return 0;
  const n = Number.parseFloat(raw);
  return Number.isFinite(n) ? n : 0;
}

/** 与链上 indexPrice 字段一致：6 位 USDC 精度 */
export function usdToIndexPrice1e6(usd: string | undefined | null): string {
  const n = quotePriceUsd(usd);
  if (n <= 0) return "0";
  return Math.round(n * 1e6).toString();
}

export function rowToQuote(row: Record<string, unknown>): MarketQuoteRow | null {
  const perp = row.perp;
  if (typeof perp !== "string" || !perp) return null;
  return {
    perp: perp.toLowerCase(),
    market_name: String(row.market_name ?? ""),
    product_id: String(row.product_id ?? ""),
    price_usd: String(row.price_usd ?? "0"),
    open_24h: String(row.open_24h ?? "0"),
    volume_24h: String(row.volume_24h ?? "0"),
    low_24h: String(row.low_24h ?? "0"),
    high_24h: String(row.high_24h ?? "0"),
    price_change_24h: String(row.price_change_24h ?? "0"),
    price_change_percent_24h: String(row.price_change_percent_24h ?? "0"),
    source: String(row.source ?? "coinbase"),
    updated_at: String(row.updated_at ?? ""),
  };
}
