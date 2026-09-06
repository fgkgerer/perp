/** 资金费率展示与结算倒计时 */

import { METANODE_API_BASE } from "@/lib/metanode-api";

export type FundingRateLatestDTO = {
  perp: string;
  periodRatePct: string;
  periodRateSource: "last_settle" | "predicted" | string;
  predictedPeriodRatePct: string;
  cumulativeRate: string;
  lastSettleAt: number;
  nextSettleAt: number;
  settleIntervalSec: number;
  settleSchedule: string;
};

export type GetLatestFundingRateRespDTO = {
  code: number;
  message: string;
  latest: FundingRateLatestDTO;
};

export async function fetchMetanodeFundingLatest(
  perp: string
): Promise<GetLatestFundingRateRespDTO> {
  const q = new URLSearchParams({ perp });
  const res = await fetch(`${METANODE_API_BASE}/api/v1/funding-rate/latest?${q}`, {
    cache: "no-store",
  });
  if (!res.ok) {
    throw new Error(`资金费接口 HTTP ${res.status}（请确认后端已重启并包含 /funding-rate/latest）`);
  }
  return (await res.json()) as GetLatestFundingRateRespDTO;
}

/** 将 periodRatePct（如 0.0123）格式化为带符号的百分比展示 */
export function formatFundingRatePct(pctStr: string): string {
  const n = Number.parseFloat(pctStr);
  if (!Number.isFinite(n)) return "—";
  const sign = n > 0 ? "+" : "";
  return `${sign}${n.toFixed(4)}%`;
}

export function fundingRateTone(pctStr: string): "positive" | "negative" | "neutral" {
  const n = Number.parseFloat(pctStr);
  if (!Number.isFinite(n) || Math.abs(n) < 1e-8) return "neutral";
  return n > 0 ? "positive" : "negative";
}

/** 格式 HH:MM:SS 倒计时 */
export function formatCountdown(msRemaining: number): string {
  if (msRemaining <= 0) return "00:00:00";
  const totalSec = Math.floor(msRemaining / 1000);
  const h = Math.floor(totalSec / 3600);
  const m = Math.floor((totalSec % 3600) / 60);
  const s = totalSec % 60;
  const pad = (x: number) => String(x).padStart(2, "0");
  return `${pad(h)}:${pad(m)}:${pad(s)}`;
}

/** 客户端 fallback：下一 UTC 8h 档（与后端 28800 配置一致） */
export function nextFundingSettleMsClient(now = Date.now()): number {
  const t = new Date(now);
  const utc = Date.UTC(
    t.getUTCFullYear(),
    t.getUTCMonth(),
    t.getUTCDate(),
    0,
    0,
    0,
    0
  );
  const hours = [0, 8, 16, 24];
  for (const h of hours) {
    const slot =
      h === 24
        ? utc + 24 * 3600 * 1000
        : utc + h * 3600 * 1000;
    if (slot > now) return slot;
  }
  return utc + 24 * 3600 * 1000;
}
