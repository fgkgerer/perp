"use client";

import { useState, useRef, useEffect } from "react";
import Link from "next/link";
import Image from "next/image";
import { formatNumber } from "@/lib/format";
import { markPriceToUsd } from "@/lib/metanode-markets";
import type { PerpMarketDTO } from "@/lib/metanode-api";
import type { RealtimeStatus } from "@/lib/useMarketQuotesRealtime";
import {
  ChevronDownIcon,
  Cog6ToothIcon,
  GlobeAltIcon,
  MagnifyingGlassIcon,
} from "@heroicons/react/24/outline";
import { ConnectButton } from "@rainbow-me/rainbowkit";

type Props = {
  markets: PerpMarketDTO[];
  selectedPerp: string;
  onSelectPerp: (address: string) => void;
  onOpenProfile?: () => void;
  indexPriceUsd?: number;
  realtimeStatus?: RealtimeStatus;
  /** 交易页由 FuturesContractBar 展示行情时隐藏顶栏重复字段 */
  compact?: boolean;
};

export default function MetaNodeTopBar({
  markets,
  selectedPerp,
  onSelectPerp,
  onOpenProfile,
  indexPriceUsd = 0,
  realtimeStatus = "disabled",
  compact = false,
}: Props) {
  const [isOpen, setIsOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  const current = markets.find(
    (m) => m.address.toLowerCase() === selectedPerp.toLowerCase()
  );
  const markUsd = markPriceToUsd(current?.markPrice);
  const indexUsd =
    indexPriceUsd > 0 ? indexPriceUsd : markPriceToUsd(current?.indexPrice);
  const live = realtimeStatus === "subscribed";
  const displayName = current?.name ?? "—";
  const currentBaseSymbol = displayName.split("-")[0].toUpperCase();
  const currentIconSrc =
    currentBaseSymbol === "BTC"
      ? "/crypto/btc.png"
      : currentBaseSymbol === "ETH"
        ? "/crypto/eth.png"
        : null;

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
    <>
      <div className="fixed left-0 right-0 top-0 z-50 flex h-6 items-center justify-center bg-accent px-4 text-center text-[10px] font-semibold tracking-wide text-[#031011]">
        MetaNode 永续合约已接入 Sepolia 测试网 · 链上透明结算 · 实时指数定价
      </div>
      <header className="fixed left-0 right-0 top-6 z-40 h-14 border-b border-panelBorder bg-[#050708]/98 px-3 md:px-5">
      <div className="mx-auto flex h-full max-w-[1920px] items-center gap-2 md:gap-5">
        <div className="mr-1 flex shrink-0 items-center gap-2.5">
          <div className="relative grid h-8 w-8 place-items-center overflow-hidden rounded bg-accent text-[13px] font-black text-page">
            M
            <span className="absolute -bottom-2 -right-2 h-4 w-4 rounded-full border border-page/30 bg-white/40" />
          </div>
          <div className="hidden leading-none sm:block">
            <div className="text-[15px] font-bold tracking-tight text-white">MetaNode</div>
            <div className="mt-1 text-[9px] font-medium uppercase tracking-[0.22em] text-subtle">Perpetual</div>
          </div>
        </div>

        <div className="hidden h-5 w-px bg-panelBorder lg:block" />

        <nav className="hidden h-full items-center gap-1 lg:flex">
          <span className="flex h-full items-center border-b-2 border-accent px-3 text-xs font-semibold text-white">永续合约</span>
          <span className="px-3 text-xs font-medium text-muted">U 本位</span>
          <span className="px-3 text-xs font-medium text-muted">链上行情</span>
          <span className="px-3 text-xs font-medium text-muted">合约信息</span>
        </nav>

        <Link
          href="/positions"
          className="hidden shrink-0 rounded px-2.5 py-2 text-xs font-medium text-muted hover:bg-elevated hover:text-white xl:block"
        >
          持仓
        </Link>
        {onOpenProfile ? (
          <button
            type="button"
            onClick={onOpenProfile}
            className="hidden shrink-0 rounded border-0 bg-transparent px-2.5 py-2 text-xs font-medium text-muted hover:bg-elevated hover:text-white xl:block"
          >
            账户
          </button>
        ) : null}

        <div ref={containerRef} className="relative shrink-0 lg:hidden">
          <button
            type="button"
            onClick={() => setIsOpen(!isOpen)}
            className="flex h-9 items-center gap-2 rounded border border-panelBorder bg-elevated/55 px-2 text-left hover:border-faint hover:bg-elevated md:min-w-[168px]"
          >
            {currentIconSrc ? (
              <Image
                src={currentIconSrc}
                alt={`${currentBaseSymbol} 图标`}
                width={28}
                height={28}
                className="h-7 w-7 rounded-full object-cover"
              />
            ) : (
              <div className="flex h-7 w-7 items-center justify-center rounded-full bg-gradient-to-br from-accent to-cyan-400 text-[11px] font-black text-page">
                {currentBaseSymbol.slice(0, 1) || "?"}
              </div>
            )}
            <div className="text-left">
              <div className="flex items-center gap-1.5">
                <div className="text-[13px] font-semibold text-white">
                  {displayName}
                </div>
                <ChevronDownIcon
                  className={`h-3.5 w-3.5 text-subtle transition ${isOpen ? "rotate-180" : ""}`}
                />
              </div>
              <div className="mt-0.5 flex items-center gap-1.5 text-[9px] text-subtle">
                <span className={`h-1.5 w-1.5 rounded-full ${live ? "bg-buy shadow-[0_0_8px_#2ecb8b]" : "bg-faint"}`} />
                Sepolia · 永续
              </div>
            </div>
          </button>

          {isOpen ? (
            <div className="absolute left-0 top-full z-50 mt-2 w-[min(90vw,420px)] overflow-hidden rounded-lg border border-panelBorder bg-panel shadow-2xl shadow-black/50">
              <div className="flex items-center justify-between border-b border-panelBorder px-3 py-2.5 text-xs text-subtle">
                <span>选择永续合约</span>
                <span className="text-[10px] text-buy">Sepolia</span>
              </div>
              <div className="max-h-[360px] overflow-y-auto">
                {markets.map((m) => {
                  const selected =
                    m.address.toLowerCase() === selectedPerp.toLowerCase();
                  return (
                    <button
                      key={m.address}
                      type="button"
                      onClick={() => {
                        onSelectPerp(m.address);
                        setIsOpen(false);
                      }}
                      className={`flex w-full items-center justify-between border-0 px-4 py-3 text-sm transition hover:bg-panelBorder ${
                        selected ? "bg-elevated" : "bg-transparent"
                      }`}
                    >
                      <span className="font-medium text-white">{m.name}</span>
                      <span className="font-mono text-muted">
                        {formatNumber(markPriceToUsd(m.markPrice), 2)}
                      </span>
                    </button>
                  );
                })}
              </div>
            </div>
          ) : null}
        </div>

        <div className="ml-auto hidden items-center gap-2 text-[10px] text-subtle xl:flex">
          <span className={`h-1.5 w-1.5 rounded-full ${live ? "bg-buy" : "bg-faint"}`} />
          {live ? "行情实时同步" : "行情轮询中"}
        </div>

        <div
          className={`hidden flex-1 flex-wrap items-center gap-8 text-sm text-muted md:flex ${compact ? "!hidden" : ""}`}
        >
          <div>
            <div className="text-xs text-subtle">标记价格</div>
            <div className="text-lg font-semibold text-accent">
              {formatNumber(markUsd, 2)}
            </div>
          </div>
          <div>
            <div className="flex items-center gap-1.5 text-xs text-subtle">
              指数价格
              {live ? (
                <span className="rounded bg-emerald-500/20 px-1.5 py-0.5 text-[10px] font-medium text-emerald-400">
                  LIVE
                </span>
              ) : null}
            </div>
            <div className="text-lg font-semibold text-foreground">
              {formatNumber(indexUsd, 2)}
            </div>
          </div>
          <div>
            <div className="text-xs text-subtle">资金费率指数</div>
            <div className="text-sm font-medium text-yellow-500">
              {current?.fundingRate && current.fundingRate !== "0"
                ? current.fundingRate
                : "—"}
            </div>
          </div>
          <div>
            <div className="text-xs text-subtle">合约</div>
            <div className="max-w-[140px] truncate font-mono text-[11px] text-muted">
              {selectedPerp.slice(0, 10)}…
            </div>
          </div>
        </div>

        <div className="hidden items-center gap-1 lg:flex">
          <button type="button" aria-label="搜索" className="grid h-8 w-8 place-items-center rounded border-0 bg-transparent text-muted hover:bg-elevated hover:text-white">
            <MagnifyingGlassIcon className="h-4 w-4" />
          </button>
          <button type="button" aria-label="语言" className="grid h-8 w-8 place-items-center rounded border-0 bg-transparent text-muted hover:bg-elevated hover:text-white">
            <GlobeAltIcon className="h-4 w-4" />
          </button>
          <button type="button" aria-label="设置" className="grid h-8 w-8 place-items-center rounded border-0 bg-transparent text-muted hover:bg-elevated hover:text-white">
            <Cog6ToothIcon className="h-4 w-4" />
          </button>
        </div>

        <div className="ml-auto hidden md:block lg:ml-0">
          <ConnectButton.Custom>
            {({
              account,
              chain,
              mounted,
              openAccountModal,
              openChainModal,
              openConnectModal,
            }) => {
              const connected = mounted && account && chain;

              if (!connected) {
                return (
                  <button
                    type="button"
                    onClick={openConnectModal}
                    className="h-9 rounded border-0 bg-accent px-4 text-xs font-bold text-[#031011] transition hover:brightness-110"
                  >
                    连接钱包
                  </button>
                );
              }

              return (
                <div className="flex items-center gap-1.5">
                  <button
                    type="button"
                    onClick={openChainModal}
                    className={
                      "flex h-8 items-center gap-1.5 rounded border border-panelBorder bg-elevated/80 px-2.5 text-[11px] font-semibold transition hover:border-faint hover:bg-elevated " +
                      (chain.unsupported ? "text-sell" : "text-white")
                    }
                  >
                    {chain.iconUrl ? (
                      <img
                        src={chain.iconUrl}
                        alt=""
                        className="h-4 w-4 rounded-full"
                      />
                    ) : (
                      <span className="h-2 w-2 rounded-full bg-accent" />
                    )}
                    <span>{chain.unsupported ? "网络错误" : chain.name}</span>
                    <ChevronDownIcon className="h-3 w-3 text-subtle" />
                  </button>

                  <button
                    type="button"
                    onClick={openAccountModal}
                    className="flex h-8 max-w-[142px] items-center gap-1.5 rounded border border-panelBorder bg-elevated/80 px-2.5 text-[11px] font-semibold text-white transition hover:border-faint hover:bg-elevated"
                  >
                    {account.ensAvatar ? (
                      <img
                        src={account.ensAvatar}
                        alt=""
                        className="h-4 w-4 rounded-full"
                      />
                    ) : (
                      <span className="grid h-4 w-4 place-items-center rounded-full bg-buy text-[8px] font-black text-[#03120d]">
                        W
                      </span>
                    )}
                    <span className="truncate">{account.displayName}</span>
                    <ChevronDownIcon className="h-3 w-3 shrink-0 text-subtle" />
                  </button>
                </div>
              );
            }}
          </ConnectButton.Custom>
        </div>
        <div className="md:hidden">
          <ConnectButton
            showBalance={false}
            accountStatus="avatar"
            chainStatus="none"
          />
        </div>
      </div>
      </header>
    </>
  );
}
