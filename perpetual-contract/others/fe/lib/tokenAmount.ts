import { formatUnits } from "viem";
import { formatNumber } from "@/lib/format";

/** viem/wagmi 对 uint8 decimals 可能返回 number 或 bigint */
export function erc20Decimals(raw: unknown, fallback = 6): number {
  if (typeof raw === "number" && Number.isFinite(raw)) return raw;
  if (typeof raw === "bigint") return Number(raw);
  return fallback;
}

/** 人类可读的 ERC20 数量（默认 USDC 6 位、展示 2 位小数） */
export function formatTokenAmount(
  raw: bigint | undefined | null,
  decimals: number,
  displayDigits = 2
): string {
  if (raw == null) return "—";
  try {
    const n = Number(formatUnits(raw, decimals));
    if (!Number.isFinite(n)) return "—";
    return formatNumber(n, displayDigits);
  } catch {
    return "—";
  }
}
