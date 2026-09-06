import type { PositionDTO } from "@/lib/metanode-api";
import { markPriceToUsd, paperToSize } from "@/lib/metanode-markets";

/** 未实现盈亏（USDC），多：(mark-entry)*size，空：(entry-mark)*size */
export function calcUnrealizedPnl(pos: PositionDTO): number {
  const size = Math.abs(paperToSize(pos.paper));
  if (size === 0) return 0;
  const entry = markPriceToUsd(pos.entryPrice);
  const mark = markPriceToUsd(pos.markPrice);
  const diff = mark - entry;
  return paperToSize(pos.paper) > 0 ? diff * size : -diff * size;
}

/** 相对开仓价的收益率（小数，如 0.01 = 1%） */
export function calcPnlRatio(pos: PositionDTO): number {
  const entry = markPriceToUsd(pos.entryPrice);
  if (entry === 0) return 0;
  const mark = markPriceToUsd(pos.markPrice);
  const raw = (mark - entry) / entry;
  return paperToSize(pos.paper) > 0 ? raw : -raw;
}

export function calcNotionalUsd(pos: PositionDTO): number {
  const size = Math.abs(paperToSize(pos.paper));
  return size * markPriceToUsd(pos.markPrice);
}

export function sumUnrealizedPnl(positions: PositionDTO[]): number {
  return positions.reduce((acc, p) => acc + calcUnrealizedPnl(p), 0);
}
