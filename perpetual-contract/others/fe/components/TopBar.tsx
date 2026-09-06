"use client";

import Link from "next/link";
import { useState, useRef, useEffect } from "react";
import { FuturesRow } from "@/lib/types";
import { formatCompact, formatNumber, formatPercent } from "@/lib/format";
import { ChevronDownIcon } from "@heroicons/react/24/outline";
import { ConnectButton } from "@rainbow-me/rainbowkit";
import MarketSelector from "./MarketSelector";

type Props = {
  symbol: string;
  market?: FuturesRow;
  markets: FuturesRow[];
  onSelect: (symbol: string) => void;
  /** 打开「我的账户」（充值 / 余额） */
  onOpenProfile?: () => void;
};

export default function TopBar({
  symbol,
  market,
  markets,
  onSelect,
  onOpenProfile,
}: Props) {
  const [isOpen, setIsOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  const change =
    market && market["24h_open"]
      ? (market["24h_close"] - market["24h_open"]) / market["24h_open"]
      : 0;
  const changeColor = change >= 0 ? "text-buy" : "text-sell";
  const displaySymbol = symbol.replace("PERP_", "").replace("_", "-");
  
  // Close when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (
        containerRef.current &&
        !containerRef.current.contains(event.target as Node)
      ) {
        setIsOpen(false);
      }
    };
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  return (
    <div className="fixed left-0 right-0 top-0 z-40 border-b border-panelBorder bg-surface px-3 py-2 md:px-6 md:py-3">
      <div className="flex items-center justify-between gap-3 md:justify-start md:gap-6">
        <Link
          href="/positions"
          className="shrink-0 rounded-lg bg-elevated px-3 py-2 text-xs font-semibold text-foreground hover:bg-panelBorder md:text-sm"
        >
          持仓
        </Link>
        {onOpenProfile ? (
          <button
            type="button"
            onClick={onOpenProfile}
            className="shrink-0 rounded-lg bg-elevated px-3 py-2 text-xs font-semibold text-foreground hover:bg-panelBorder md:text-sm"
          >
            账户
          </button>
        ) : null}

        {/* Symbol Selector Trigger */}
        <div ref={containerRef} className="relative shrink-0">
            <button 
                onClick={() => setIsOpen(!isOpen)}
                className="flex items-center gap-2 rounded-lg p-1 transition bg-elevated border-0 hover:bg-panelBorder md:gap-3 md:p-2 md:-ml-2"
            >
            <div className="flex h-8 w-8 items-center justify-center rounded-full bg-accent text-center text-xs font-bold leading-8 text-black md:h-10 md:w-10 md:text-sm md:leading-10">
                {displaySymbol.substring(0, 1)}
            </div>
            <div className="text-left">
                <div className="flex items-center gap-1 md:gap-2">
                    <div className="text-sm font-bold text-white md:text-lg">{displaySymbol}</div>
                    <ChevronDownIcon className={`h-3 w-3 text-muted transition md:h-4 md:w-4 ${isOpen ? 'rotate-180' : ''}`} />
                </div>
                <div className="text-[10px] text-muted md:text-xs">永续合约</div>
            </div>
            </button>
            
            {/* Dropdown Menu */}
            {isOpen && (
                <MarketSelector 
                    markets={markets} 
                    selected={symbol} 
                    onSelect={onSelect} 
                    onClose={() => setIsOpen(false)}
                />
            )}
        </div>

        {/* Mobile Price Stats (Compact) */}
        <div className="flex flex-col items-end md:hidden">
            <div className={`text-sm font-semibold ${changeColor}`}>
              {formatNumber(market?.mark_price ?? 0, 2)}
            </div>
            <div className={`text-[10px] ${changeColor}`}>
               {formatPercent(change)}
            </div>
        </div>

        {/* Desktop Market Stats */}
        <div className="hidden flex-1 flex-wrap items-center gap-8 text-sm text-muted md:flex">
          <div>
            <div className={`text-lg font-semibold ${changeColor}`}>
              {formatNumber(market?.mark_price ?? 0, 2)}
            </div>
            <div className={`text-xs ${changeColor}`}>
               {formatPercent(change)}
            </div>
          </div>
          <div>
            <div className="text-xs text-muted">标记价格</div>
            <div className="text-sm font-medium text-foreground">
               {formatNumber(market?.mark_price ?? 0, 2)}
            </div>
          </div>
          <div>
            <div className="text-xs text-muted">指数价格</div>
            <div className="text-sm font-medium text-foreground">
              {formatNumber(market?.index_price ?? 0, 2)}
            </div>
          </div>
          <div>
            <div className="text-xs text-muted">24h 最高 / 最低</div>
            <div className="text-sm font-medium text-foreground">
              {formatNumber(market?.["24h_high"] ?? 0, 2)} /{" "}
              {formatNumber(market?.["24h_low"] ?? 0, 2)}
            </div>
          </div>
          <div>
            <div className="text-xs text-muted">24h 成交额(USDC)</div>
            <div className="text-sm font-medium text-foreground">
              {formatCompact(market?.["24h_amount"] ?? 0)}
            </div>
          </div>
          <div>
            <div className="text-xs text-muted">资金费率</div>
            <div className="text-sm font-medium text-warning text-yellow-500">
              {formatPercent(market?.est_funding_rate ?? 0, 4)}%
            </div>
          </div>
        </div>

        <div className="hidden md:ml-auto md:block">
            <ConnectButton />
        </div>
        <div className="md:hidden">
            <ConnectButton showBalance={false} accountStatus="avatar" chainStatus="none" />
        </div>
      </div>
    </div>
  );
}
