"use client";

import { useMemo, useState } from "react";
import { ChevronDownIcon } from "@heroicons/react/20/solid";
import { formatNumber } from "@/lib/format";
import { markPriceToUsd, paperToSize } from "@/lib/metanode-markets";
import type { OrderBookEntryDTO } from "@/lib/metanode-api";

type Level = { price: number; amount: number };
type BookRow = Level & { total: number };
type ViewMode = "both" | "bids" | "asks";

type Props = {
  bids: OrderBookEntryDTO[];
  asks: OrderBookEntryDTO[];
  loading?: boolean;
  markPriceUsd?: number;
  symbol?: string;
  /** 嵌入 Tab 面板时不重复外框标题 */
  embedded?: boolean;
};

function toLevels(entries: OrderBookEntryDTO[]): Level[] {
  return entries.map((entry) => ({
    price: markPriceToUsd(entry.price),
    amount: Math.abs(paperToSize(entry.amount)),
  }));
}

function withCumulative(levels: Level[]): BookRow[] {
  let sum = 0;
  return levels.map((row) => {
    sum += row.amount;
    return { ...row, total: sum };
  });
}

function formatBookPrice(price: number, precision: number): string {
  return formatNumber(price, precision < 1 ? 1 : 0);
}

export default function OrderBook({
  bids,
  asks,
  loading = false,
  markPriceUsd = 0,
  symbol = "BTC",
  embedded = false,
}: Props) {
  const [viewMode, setViewMode] = useState<ViewMode>("both");
  const [precision, setPrecision] = useState(0.1);

  const { askRows, bidRows, bidSharePct, maxTotal } = useMemo(() => {
    const asksWithTotal = withCumulative(
      toLevels(asks).filter((row) => row.price > 0 && row.amount > 0)
    );
    const bidsWithTotal = withCumulative(
      toLevels(bids).filter((row) => row.price > 0 && row.amount > 0)
    );
    const askSum = asksWithTotal.at(-1)?.total ?? 0;
    const bidSum = bidsWithTotal.at(-1)?.total ?? 0;
    const total = askSum + bidSum;

    return {
      askRows: asksWithTotal,
      bidRows: bidsWithTotal,
      bidSharePct: total > 0 ? (bidSum / total) * 100 : 50,
      maxTotal: Math.max(askSum, bidSum, 1),
    };
  }, [asks, bids]);

  const rowLimit = viewMode === "both" ? 6 : 15;
  const visibleAsks = askRows.slice(0, rowLimit).reverse();
  const visibleBids = bidRows.slice(0, rowLimit);
  const hasRows = askRows.length > 0 || bidRows.length > 0;

  const body = (
    <div className="flex h-full min-h-0 flex-col bg-[#070a0b] font-mono">
      <div className="flex h-11 shrink-0 items-center justify-between px-4">
        <div className="flex items-center gap-3">
          <BookViewButton
            mode="both"
            active={viewMode === "both"}
            label="同时显示买卖盘"
            onClick={() => setViewMode("both")}
          />
          <BookViewButton
            mode="bids"
            active={viewMode === "bids"}
            label="仅显示买盘"
            onClick={() => setViewMode("bids")}
          />
          <BookViewButton
            mode="asks"
            active={viewMode === "asks"}
            label="仅显示卖盘"
            onClick={() => setViewMode("asks")}
          />
        </div>

        <label className="relative flex h-7 items-center rounded bg-[#2b3038] pl-2.5 pr-7 text-[11px] font-semibold text-white hover:bg-[#343a43]">
          {precision.toFixed(1)}
          <ChevronDownIcon className="absolute right-2 h-3.5 w-3.5 text-subtle" />
          <select
            aria-label="订单簿价格精度"
            value={precision}
            onChange={(event) => setPrecision(Number(event.target.value))}
            className="absolute inset-0 cursor-pointer opacity-0"
          >
            <option value={0.1}>0.1</option>
            <option value={1}>1</option>
            <option value={10}>10</option>
          </select>
        </label>
      </div>

      <div className="grid h-8 shrink-0 grid-cols-[1.05fr_0.85fr_1fr] items-center px-4 font-sans text-[10px] text-subtle">
        <span>单价(USDT)</span>
        <span className="text-right">数量({symbol})</span>
        <span className="text-right">累计({symbol})</span>
      </div>

      <div className="relative min-h-0 flex-1 overflow-hidden">
        {viewMode !== "bids" ? (
          <BookRows
            rows={visibleAsks}
            side="ask"
            maxTotal={maxTotal}
            precision={precision}
            loading={loading}
            empty={!hasRows}
            count={rowLimit}
          />
        ) : null}

        {viewMode === "both" ? (
          <div className="flex h-12 shrink-0 items-center gap-2 px-4">
            <span
              key={markPriceUsd}
              className="last-price-tick text-xl font-bold tracking-tight text-buy"
            >
              {markPriceUsd > 0 ? formatBookPrice(markPriceUsd, precision) : "—"}
            </span>
            <span className="text-base text-buy">↑</span>
            <span className="border-b border-dashed border-faint text-[11px] text-subtle">
              {markPriceUsd > 0 ? formatBookPrice(markPriceUsd, precision) : "—"}
            </span>
          </div>
        ) : null}

        {viewMode !== "asks" ? (
          <BookRows
            rows={visibleBids}
            side="bid"
            maxTotal={maxTotal}
            precision={precision}
            loading={loading}
            empty={!hasRows}
            count={rowLimit}
          />
        ) : null}
        {!loading && !hasRows ? (
          <span className="pointer-events-none absolute inset-0 grid place-items-center font-sans text-[10px] text-subtle">
            暂无挂单
          </span>
        ) : null}
      </div>

      <div className="flex h-9 shrink-0 items-center px-4 font-sans text-[11px] font-semibold">
        <div className="relative flex h-5 w-full overflow-hidden bg-sell/15">
          <div
            className="book-ratio-buy flex items-center bg-buy/25 pl-2 text-buy transition-[width] duration-500 ease-out"
            style={{ width: `${bidSharePct}%` }}
          >
            买&nbsp;&nbsp;{formatNumber(bidSharePct, 2)}%
          </div>
          <div className="flex flex-1 items-center justify-end bg-sell/25 pr-2 text-sell">
            {formatNumber(100 - bidSharePct, 2)}%&nbsp;&nbsp;卖
          </div>
        </div>
      </div>
    </div>
  );

  if (embedded) return body;

  return (
    <div className="panel overflow-hidden">
      <div className="panel-header">订单簿</div>
      {body}
    </div>
  );
}

function BookRows({
  rows,
  side,
  maxTotal,
  precision,
  loading,
  empty,
  count,
}: {
  rows: BookRow[];
  side: "bid" | "ask";
  maxTotal: number;
  precision: number;
  loading: boolean;
  empty: boolean;
  count: number;
}) {
  if (loading || empty) {
    return (
      <div className="relative">
        {Array.from({ length: count }).map((_, index) => (
          <div
            key={`${side}-placeholder-${index}`}
            className="grid h-[22px] grid-cols-[1.05fr_0.85fr_1fr] items-center px-4 text-[11px] text-faint"
          >
            <span className={side === "bid" ? "text-buy/25" : "text-sell/25"}>—</span>
            <span className="text-right">—</span>
            <span className="text-right">—</span>
          </div>
        ))}
      </div>
    );
  }

  return (
    <div>
      {rows.map((row) => {
        const depth = Math.max(4, Math.min(100, (row.total / maxTotal) * 100));
        return (
          <div
            key={`${side}-${row.price}-${row.total}`}
            className={`group relative grid h-[22px] grid-cols-[1.05fr_0.85fr_1fr] items-center px-4 text-[11px] ${
              side === "bid" ? "book-row-bid" : "book-row-ask"
            }`}
          >
            <span
              className={`absolute inset-y-px right-0 transition-[width] duration-300 ease-out ${
                side === "bid" ? "bg-buy/[0.16]" : "bg-sell/[0.16]"
              }`}
              style={{ width: `${depth}%` }}
            />
            <span className={`relative z-10 ${side === "bid" ? "text-buy" : "text-sell"}`}>
              {formatBookPrice(row.price, precision)}
            </span>
            <span className="relative z-10 text-right text-foreground">
              {formatNumber(row.amount, 3)}
            </span>
            <span className="relative z-10 text-right text-foreground">
              {formatNumber(row.total, 3)}
            </span>
          </div>
        );
      })}
    </div>
  );
}

function BookViewButton({
  mode,
  active,
  label,
  onClick,
}: {
  mode: ViewMode;
  active: boolean;
  label: string;
  onClick: () => void;
}) {
  const colors =
    mode === "both"
      ? ["bg-sell", "bg-buy"]
      : mode === "bids"
        ? ["bg-faint", "bg-buy"]
        : ["bg-sell", "bg-faint"];

  return (
    <button
      type="button"
      aria-label={label}
      aria-pressed={active}
      onClick={onClick}
      className={`grid h-7 w-7 place-items-center rounded-sm border-0 ${
        active ? "bg-[#2b3038]" : "bg-transparent hover:bg-elevated"
      }`}
    >
      <span className="grid w-3.5 gap-1">
        <span className={`h-0.5 w-full ${colors[0]}`} />
        <span className={`h-0.5 w-full ${colors[1]}`} />
      </span>
    </button>
  );
}

/** 买盘数量占比，供合约信息条展示 */
export function orderBookBidShare(
  bids: OrderBookEntryDTO[],
  asks: OrderBookEntryDTO[]
): number | undefined {
  let bidSum = 0;
  let askSum = 0;
  for (const entry of bids) bidSum += Math.abs(paperToSize(entry.amount));
  for (const entry of asks) askSum += Math.abs(paperToSize(entry.amount));
  const total = bidSum + askSum;
  if (total <= 0) return undefined;
  return (bidSum / total) * 100;
}
