"use client";

import { formatNumber, formatTime } from "@/lib/format";

export type DisplayTrade = {
  price: number;
  size: number;
  ts: number;
  side: "BUY" | "SELL";
  txHash?: string;
};

type Props = {
  trades: DisplayTrade[];
  loading?: boolean;
  embedded?: boolean;
  symbol?: string;
};

export default function Trades({
  trades,
  loading = false,
  embedded = false,
  symbol = "BTC",
}: Props) {
  const body = (
    <div className="flex h-full min-h-0 flex-col bg-[#070a0b] font-mono">
      <div className="grid h-10 shrink-0 grid-cols-[1fr_0.9fr_0.9fr] items-center border-b border-panelBorder px-4 font-sans text-[10px] text-subtle">
        <span>价格(USDT)</span>
        <span className="text-right">数量({symbol})</span>
        <span className="text-right">时间</span>
      </div>

      <div className="terminal-scrollbar min-h-0 flex-1 overflow-y-auto py-1">
        {loading ? (
          <TradePlaceholders />
        ) : trades.length > 0 ? (
          trades.slice(0, 30).map((trade, index) => (
            <div
              key={`${trade.ts}-${trade.price}-${trade.size}-${index}`}
              className={`trade-row-new grid h-6 grid-cols-[1fr_0.9fr_0.9fr] items-center px-4 text-[11px] hover:bg-white/[0.025] ${
                trade.side === "BUY" ? "trade-row-buy" : "trade-row-sell"
              }`}
              title={trade.txHash ? `交易哈希：${trade.txHash}` : undefined}
            >
              <span className={trade.side === "BUY" ? "text-buy" : "text-sell"}>
                {formatNumber(trade.price, 1)}
              </span>
              <span className="text-right text-foreground">
                {formatNumber(trade.size, 3)}
              </span>
              <span className="text-right text-subtle">
                {formatTime(trade.ts * 1000)}
              </span>
            </div>
          ))
        ) : (
          <div className="relative h-full min-h-[350px]">
            <TradePlaceholders muted />
            <span className="absolute inset-0 grid place-items-center font-sans text-[10px] text-subtle">
              暂无链上成交记录
            </span>
          </div>
        )}
      </div>

      <div className="flex h-8 shrink-0 items-center justify-between border-t border-panelBorder px-4 font-sans text-[9px] text-subtle">
        <span className="flex items-center gap-1.5">
          <span className="h-1.5 w-1.5 rounded-full bg-buy shadow-[0_0_6px_#2ecb8b]" />
          实时逐笔成交
        </span>
        <span>链上撮合</span>
      </div>
    </div>
  );

  if (embedded) return body;

  return (
    <div className="panel overflow-hidden">
      <div className="panel-header">最新成交</div>
      {body}
    </div>
  );
}

function TradePlaceholders({ muted = false }: { muted?: boolean }) {
  return (
    <div className={muted ? "opacity-40" : "animate-pulse opacity-60"}>
      {Array.from({ length: 16 }).map((_, index) => (
        <div
          key={`trade-placeholder-${index}`}
          className="grid h-6 grid-cols-[1fr_0.9fr_0.9fr] items-center px-4 text-[11px] text-faint"
        >
          <span className={index % 3 === 0 ? "text-sell/25" : "text-buy/25"}>—</span>
          <span className="text-right">—</span>
          <span className="text-right">—</span>
        </div>
      ))}
    </div>
  );
}
