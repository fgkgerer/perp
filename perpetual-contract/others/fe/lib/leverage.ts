import type { PerpMarketDTO } from "@/lib/metanode-api";

/** Sepolia 部署默认（与 script/DeploySepolia.s.sol 一致） */
export const SEPOLIA_DEFAULT_RISK: Record<
  string,
  { maxLeverage: string; initialMarginPct: string; maintenanceMarginPct: string }
> = {
  "0x11aae1f92ff10bfbb205971e060cf6d9d917723b": {
    maxLeverage: "20",
    initialMarginPct: "5.00",
    maintenanceMarginPct: "3.00",
  },
  "0x98456dcbcefea550293727a7e2dfd45de92740c0": {
    maxLeverage: "10",
    initialMarginPct: "10.00",
    maintenanceMarginPct: "5.00",
  },
};

export function resolveMarketRisk(market: PerpMarketDTO | undefined) {
  if (!market) {
    return { maxLeverage: "", initialMarginPct: "", maintenanceMarginPct: "" };
  }
  const key = market.address.toLowerCase();
  const fallback = SEPOLIA_DEFAULT_RISK[key];
  return {
    maxLeverage: market.maxLeverage || fallback?.maxLeverage || "",
    initialMarginPct:
      market.initialMarginPct || fallback?.initialMarginPct || "",
    maintenanceMarginPct:
      market.maintenanceMarginPct || fallback?.maintenanceMarginPct || "",
  };
}

export function formatLeverageLabel(maxLeverage: string): string {
  const n = Number.parseFloat(maxLeverage);
  if (!Number.isFinite(n) || n <= 0) return "—";
  return Number.isInteger(n) ? `${n}x` : `${maxLeverage}x`;
}

export function parseMaxLeverage(maxLeverage: string): number {
  const n = Number.parseFloat(maxLeverage);
  if (!Number.isFinite(n) || n < 1) return 20;
  return Math.floor(n);
}

/** 常用杠杆档位（不超过链上 maxLeverage） */
const LEVERAGE_PRESETS = [1, 2, 3, 5, 10, 15, 20, 25, 50, 75, 100, 125];

export function buildLeverageOptions(maxLeverage: number): number[] {
  const max = Math.max(1, Math.floor(maxLeverage));
  const opts = LEVERAGE_PRESETS.filter((x) => x <= max);
  if (!opts.includes(max)) {
    opts.push(max);
  }
  return opts.sort((a, b) => a - b);
}

export function clampLeverage(leverage: number, maxLeverage: number): number {
  const max = Math.max(1, Math.floor(maxLeverage));
  const lev = Math.floor(leverage);
  if (!Number.isFinite(lev) || lev < 1) return 1;
  return Math.min(lev, max);
}

/** 用户所选杠杆下的开仓保证金 ≈ 名义价值 / 杠杆 */
export function marginRequiredUsd(notionalUsd: number, leverage: number): number {
  if (notionalUsd <= 0 || leverage < 1) return 0;
  return notionalUsd / leverage;
}

/** 链上 initialMarginPct 对应的最低保证金（名义 × 比例） */
export function chainMinMarginUsd(
  notionalUsd: number,
  initialMarginPct: string
): number {
  const pct = Number.parseFloat(initialMarginPct);
  if (!Number.isFinite(pct) || pct <= 0 || notionalUsd <= 0) return 0;
  return (notionalUsd * pct) / 100;
}

const LEVERAGE_STORAGE_PREFIX = "metanode_leverage_";

export function loadStoredLeverage(perp: string, maxLeverage: number): number {
  if (typeof window === "undefined") return defaultLeverage(maxLeverage);
  try {
    const raw = localStorage.getItem(`${LEVERAGE_STORAGE_PREFIX}${perp.toLowerCase()}`);
    if (!raw) return defaultLeverage(maxLeverage);
    return clampLeverage(Number.parseInt(raw, 10), maxLeverage);
  } catch {
    return defaultLeverage(maxLeverage);
  }
}

export function saveStoredLeverage(perp: string, leverage: number): void {
  if (typeof window === "undefined") return;
  try {
    localStorage.setItem(
      `${LEVERAGE_STORAGE_PREFIX}${perp.toLowerCase()}`,
      String(Math.floor(leverage))
    );
  } catch {
    /* ignore quota */
  }
}

function defaultLeverage(maxLeverage: number): number {
  const max = Math.max(1, Math.floor(maxLeverage));
  return Math.min(10, max);
}
