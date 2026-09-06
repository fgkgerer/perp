"use client";

import { useState, type ReactNode } from "react";
import { EllipsisHorizontalIcon } from "@heroicons/react/24/outline";
import OrderBook from "@/components/OrderBook";
import Trades, { type DisplayTrade } from "@/components/Trades";
import type { OrderBookEntryDTO } from "@/lib/metanode-api";

type Tab = "book" | "trades";

type Props = {
  bids: OrderBookEntryDTO[];
  asks: OrderBookEntryDTO[];
  trades: DisplayTrade[];
  bookLoading?: boolean;
  tradesLoading?: boolean;
  markPriceUsd?: number;
  symbol?: string;
  className?: string;
};

export default function MarketDepthTabs({
  bids,
  asks,
  trades,
  bookLoading,
  tradesLoading,
  markPriceUsd = 0,
  symbol = "BTC",
  className = "",
}: Props) {
  const [tab, setTab] = useState<Tab>("book");

  return (
    <div
      className={`panel flex flex-col overflow-hidden ${className}`.trim()}
    >
      <div className="flex h-12 shrink-0 items-center border-b border-panelBorder px-3">
        <div className="flex h-full items-center gap-1">
          <TabButton active={tab === "book"} onClick={() => setTab("book")}>
            订单簿
          </TabButton>
          <TabButton active={tab === "trades"} onClick={() => setTab("trades")}>
            最新成交
          </TabButton>
        </div>
        <button
          type="button"
          aria-label="更多盘口设置"
          className="ml-auto grid h-7 w-7 place-items-center rounded border-0 bg-transparent text-muted hover:bg-elevated hover:text-white"
        >
          <EllipsisHorizontalIcon className="h-5 w-5" />
        </button>
      </div>
      <div className="min-h-0 flex-1 overflow-hidden">
        {tab === "book" ? (
          <div key="book" className="market-tab-panel h-full">
            <OrderBook
              bids={bids}
              asks={asks}
              loading={bookLoading}
              markPriceUsd={markPriceUsd}
              symbol={symbol}
              embedded
            />
          </div>
        ) : (
          <div key="trades" className="market-tab-panel h-full">
            <Trades
              trades={trades}
              loading={tradesLoading}
              symbol={symbol}
              embedded
            />
          </div>
        )}
      </div>
    </div>
  );
}

function TabButton({
  active,
  onClick,
  children,
}: {
  active: boolean;
  onClick: () => void;
  children: ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`relative h-full border-0 bg-transparent px-3 text-[15px] font-semibold transition ${
        active
          ? "text-white"
          : "text-subtle hover:text-foreground"
      }`}
    >
      {children}
    </button>
  );
}
