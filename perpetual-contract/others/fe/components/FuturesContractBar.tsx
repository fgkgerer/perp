"use client";

import Image from "next/image";
import FundingRatePanel from "@/components/FundingRatePanel";
import { formatNumber } from "@/lib/format";
import { formatLeverageLabel, resolveMarketRisk } from "@/lib/leverage";
import { markPriceToUsd } from "@/lib/metanode-markets";
import type { PerpMarketDTO } from "@/lib/metanode-api";
import { ChevronDownIcon } from "@heroicons/react/24/outline";

type Props = {
  market: PerpMarketDTO | undefined;
  markets?: PerpMarketDTO[];
  selectedPerp?: string;
  onSelectPerp?: (address: string) => void;
  indexPriceUsd: number;
  /** 订单簿多空数量比（0–100，无数据为 undefined） */
  bidSharePct?: number;
};

const CRYPTO_ICON_BY_SYMBOL: Record<string, string> = {
  BTC: "/crypto/btc.png",
  ETH: "/crypto/eth.png",
};

export default function FuturesContractBar({
  market,
  markets = [],
  selectedPerp,
  onSelectPerp,
  indexPriceUsd,
  bidSharePct,
}: Props) {
  const markUsd = markPriceToUsd(market?.markPrice);
  const indexUsd =
    indexPriceUsd > 0 ? indexPriceUsd : markPriceToUsd(market?.indexPrice);
  const lastUsd = indexUsd > 0 ? indexUsd : markUsd;
  const spread =
    markUsd > 0 && indexUsd > 0
      ? Math.abs(markUsd - indexUsd)
      : 0;
  const risk = resolveMarketRisk(market);
  const leverageLabel = formatLeverageLabel(risk.maxLeverage);
  const baseSymbol = (market?.name ?? "").split("-")[0].toUpperCase();
  const iconSrc = CRYPTO_ICON_BY_SYMBOL[baseSymbol];

  return (
    <div className="border-b border-panelBorder bg-surface px-3 sm:px-5">
      <div className="mx-auto flex max-w-[1920px] flex-wrap items-stretch gap-x-6 lg:flex-nowrap">
        <div className="relative flex min-w-[235px] items-center gap-3 border-panelBorder py-2 lg:border-r lg:pr-6">
          {markets.length > 0 && selectedPerp && onSelectPerp ? (
            <select
              aria-label="选择永续合约"
              value={selectedPerp}
              onChange={(event) => onSelectPerp(event.target.value)}
              className="absolute inset-0 z-10 h-full w-full cursor-pointer opacity-0"
            >
              {markets.map((item) => (
                <option key={item.address} value={item.address}>
                  {item.name}
                </option>
              ))}
            </select>
          ) : null}
          {iconSrc ? (
            <Image
              src={iconSrc}
              alt={`${baseSymbol} 图标`}
              width={32}
              height={32}
              className="h-8 w-8 shrink-0 rounded-full object-cover"
              priority
            />
          ) : (
            <div className="grid h-8 w-8 shrink-0 place-items-center rounded-full bg-elevated text-[11px] font-black text-accent">
              {baseSymbol.slice(0, 1) || "?"}
            </div>
          )}
          <div className="min-w-0">
            <div className="flex items-center gap-2">
              <h1 className="text-sm font-bold text-white sm:text-base">
                {market?.name ?? "—"}
              </h1>
              <span className="rounded bg-accent/10 px-1.5 py-0.5 text-[9px] font-semibold text-accent">
                {leverageLabel}
              </span>
              <ChevronDownIcon className="h-3.5 w-3.5 text-subtle" />
            </div>
            <div className="mt-1 flex items-center gap-2 text-[10px] text-subtle">
              <span>USDC 保证金</span>
              <span className="h-1 w-1 rounded-full bg-faint" />
              <span className="text-buy">Sepolia</span>
            </div>
          </div>
        </div>

        <div className="flex min-w-[180px] items-center py-2">
          <div>
            <div className="flex items-baseline gap-2">
            <span
              className={`font-mono text-xl font-semibold tracking-tight sm:text-2xl ${
                lastUsd > 0 ? "text-buy" : "text-subtle"
              }`}
            >
              {lastUsd > 0 ? formatNumber(lastUsd, 2) : "—"}
            </span>
              <span className="text-[10px] text-subtle">USDT</span>
            </div>
            <p className="mt-0.5 text-[10px] text-subtle">最新指数价</p>
          </div>
        </div>

        <div className="terminal-scrollbar flex min-w-0 flex-1 items-center gap-6 overflow-x-auto border-t border-panelBorder py-2 text-xs sm:gap-8 lg:border-t-0">
          <Stat
            label="最大杠杆"
            value={leverageLabel}
            accent={leverageLabel !== "—"}
          />
          <Stat
            label="初始保证金"
            value={
              risk.initialMarginPct ? `${risk.initialMarginPct}%` : "—"
            }
          />
          <Stat
            label="维持保证金"
            value={
              risk.maintenanceMarginPct
                ? `${risk.maintenanceMarginPct}%`
                : "—"
            }
          />
          <Stat label="标记价格" value={formatNumber(markUsd, 2)} accent />
          <Stat label="指数价格" value={formatNumber(indexUsd, 2)} />
          <Stat
            label="标记 − 指数"
            value={spread > 0 ? formatNumber(spread, 2) : "—"}
          />
          {market?.address ? (
            <FundingRatePanel perp={market.address} compact />
          ) : null}
          {bidSharePct != null ? (
            <Stat
              label="盘口多空"
              value={`买 ${formatNumber(bidSharePct, 0)}% / 卖 ${formatNumber(100 - bidSharePct, 0)}%`}
            />
          ) : null}
        </div>
      </div>
    </div>
  );
}

function Stat({
  label,
  value,
  accent,
  muted,
}: {
  label: string;
  value: string;
  accent?: boolean;
  muted?: boolean;
}) {
  return (
    <div className="shrink-0 whitespace-nowrap">
      <div className="text-[10px] text-subtle">{label}</div>
      <div
        className={`mt-1 font-mono text-xs font-medium ${
          accent ? "text-accent" : muted ? "text-yellow-500" : "text-foreground"
        }`}
      >
        {value}
      </div>
    </div>
  );
}
