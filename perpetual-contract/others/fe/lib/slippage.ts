export type OrderSide = "long" | "short";

export const SLIPPAGE_PRESETS_BPS = [10, 50, 100, 300] as const;
export const DEFAULT_SLIPPAGE_BPS = 50;
export const MAX_SLIPPAGE_BPS = 5000;
export const SLIPPAGE_STORAGE_KEY = "metanode_slippage_bps";

export function normalizeSlippageBps(bps: number): number {
  if (!Number.isFinite(bps) || bps <= 0) return DEFAULT_SLIPPAGE_BPS;
  return Math.min(Math.round(bps), MAX_SLIPPAGE_BPS);
}

export function loadStoredSlippageBps(): number {
  if (typeof window === "undefined") return DEFAULT_SLIPPAGE_BPS;
  const raw = localStorage.getItem(SLIPPAGE_STORAGE_KEY);
  if (!raw) return DEFAULT_SLIPPAGE_BPS;
  return normalizeSlippageBps(Number.parseInt(raw, 10));
}

export function saveStoredSlippageBps(bps: number): void {
  if (typeof window === "undefined") return;
  localStorage.setItem(SLIPPAGE_STORAGE_KEY, String(normalizeSlippageBps(bps)));
}

/** 由参考价与滑点计算 EIP-712 签名限价（USD 字符串，2 位小数）。 */
export function applySlippageLimitPriceUsd(
  referenceUsd: string,
  side: OrderSide,
  slippageBps: number
): string {
  const ref = Number(referenceUsd);
  if (!Number.isFinite(ref) || ref <= 0) return referenceUsd;
  const bps = normalizeSlippageBps(slippageBps);
  const factor =
    side === "long" ? 1 + bps / 10_000 : 1 - bps / 10_000;
  const limit = ref * factor;
  if (!Number.isFinite(limit) || limit <= 0) return referenceUsd;
  return limit.toFixed(2);
}

export function slippagePercentLabel(bps: number): string {
  return `${(normalizeSlippageBps(bps) / 100).toFixed(2)}%`;
}

/** 手动限价相对参考价是否超出滑点容忍（用于提交前警告）。 */
export function manualPriceExceedsSlippage(
  manualUsd: string,
  referenceUsd: string,
  side: OrderSide,
  slippageBps: number
): boolean {
  const manual = Number(manualUsd);
  const ref = Number(referenceUsd);
  if (!Number.isFinite(manual) || !Number.isFinite(ref) || ref <= 0) {
    return false;
  }
  const cap = Number(
    applySlippageLimitPriceUsd(referenceUsd, side, slippageBps)
  );
  if (!Number.isFinite(cap)) return false;
  return side === "long" ? manual > cap + 1e-9 : manual < cap - 1e-9;
}
