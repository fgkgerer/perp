"use client";

import { useMemo, useState } from "react";
import { FuturesRow } from "@/lib/types";
import { formatNumber, formatPercent, formatCompact } from "@/lib/format";
import { MagnifyingGlassIcon } from "@heroicons/react/24/outline";

type Props = {
  markets: FuturesRow[];
  selected: string;
  onSelect: (symbol: string) => void;
  onClose: () => void;
};

export default function MarketSelector({
  markets,
  selected,
  onSelect,
  onClose
}: Props) {
  const [query, setQuery] = useState("");

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    const rows = q
      ? markets.filter((row) => row.symbol.toLowerCase().includes(q))
      : markets;
    return rows;
  }, [markets, query]);

  return (
    <div className="absolute left-0 top-full z-50 mt-2 flex h-[600px] w-[90vw] max-w-[700px] flex-col rounded-lg border border-panelBorder bg-[#111a2b] shadow-2xl">
      {/* Search Bar */}
      <div className="border-b border-panelBorder p-3">
        <div className="relative">
          <MagnifyingGlassIcon className="absolute left-3 top-2.5 h-4 w-4 text-muted" />
          <input
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="搜索"
            className="w-full rounded bg-elevated py-2 pl-9 pr-4 text-sm text-white outline-none placeholder:text-muted focus:ring-1 focus:ring-accent"
            autoFocus
          />
        </div>
      </div>

      {/* Header */}
      <div className="grid grid-cols-4 gap-2 px-4 py-2 text-xs font-medium text-muted">
        <span className="col-span-2 sm:col-span-1">交易对</span>
        <span className="text-right">标记价格</span>
        <span className="hidden text-right sm:block">24h 交易量</span>
        <span className="text-right">资金费率</span>
      </div>

      {/* List */}
      <div className="flex-1 overflow-y-auto">
        {filtered.map((row) => {
          const change =
            row["24h_open"] !== 0
              ? (row["24h_close"] - row["24h_open"]) / row["24h_open"]
              : 0;
          const changeColor = change >= 0 ? "text-buy" : "text-sell";
          
          return (
            <button
              key={row.symbol}
              onClick={() => {
                onSelect(row.symbol);
                onClose();
              }}
              className={`grid w-full grid-cols-4 items-center border-0 gap-2 px-4 py-3 text-sm transition hover:bg-panelBorder ${
                selected === row.symbol ? "bg-elevated" : "bg-[#1e293b]"
              }`}
            >
              {/* Symbol */}
              <div className="col-span-2 flex items-center gap-3 sm:col-span-1">
                 <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-slate-700 text-[10px] text-white">
                    {/* Simple icon placeholder based on symbol */}
                    {row.symbol.split("_")[1]?.substring(0, 1) || "?"}
                 </div>
                 <div className="flex flex-col items-start overflow-hidden">
                    <span className="truncate font-bold text-white">{row.symbol.split("_")[1]}-{row.symbol.split("_")[2]}</span>
                    <span className="text-xs text-muted">永续合约</span>
                 </div>
              </div>

              {/* Price */}
              <div className="flex flex-col items-end">
                <span className={`font-medium ${changeColor}`}>
                  {formatNumber(row.mark_price, row.mark_price < 1 ? 4 : 2)}
                </span>
                <span className={`text-xs ${changeColor}`}>
                  {formatPercent(change)}
                </span>
              </div>

              {/* Volume */}
              <div className="hidden text-right text-foreground sm:block">
                {formatCompact(row["24h_amount"])}
              </div>

              {/* Funding */}
              <div className="text-right text-foreground">
                 {formatPercent(row.est_funding_rate, 4)}%
              </div>
            </button>
          );
        })}
        {!filtered.length && (
          <div className="py-8 text-center text-muted">无匹配结果</div>
        )}
      </div>
    </div>
  );
}
