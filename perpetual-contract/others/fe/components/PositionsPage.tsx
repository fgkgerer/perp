"use client";

import Link from "next/link";
import { useCallback, useEffect, useMemo, useState } from "react";
import { ConnectButton } from "@rainbow-me/rainbowkit";
import { formatUnits } from "viem";
import { useAccount } from "wagmi";
import {
  fetchMetanodePositions,
  fetchMetanodeRisk,
  type PositionDTO,
  type RiskInfoDTO,
} from "@/lib/metanode-api";
import { markPriceToUsd, paperToSize } from "@/lib/metanode-markets";
import {
  calcNotionalUsd,
  calcPnlRatio,
  calcUnrealizedPnl,
  sumUnrealizedPnl,
} from "@/lib/position-pnl";
import { formatNumber, formatPercent } from "@/lib/format";

const USDC_DECIMALS = 6;
const REFRESH_MS = 15_000;

function formatUsdc(raw: string): string {
  try {
    return formatUnits(BigInt(raw.trim() || "0"), USDC_DECIMALS);
  } catch {
    return raw;
  }
}

function pnlClass(value: number): string {
  if (value > 0) return "text-buy";
  if (value < 0) return "text-sell";
  return "text-subtle";
}

function PositionRow({ pos }: { pos: PositionDTO }) {
  const size = paperToSize(pos.paper);
  const isLong = size > 0;
  const pnl = calcUnrealizedPnl(pos);
  const pnlRatio = calcPnlRatio(pos);
  const notional = calcNotionalUsd(pos);
  const liq = markPriceToUsd(pos.liqPrice);

  return (
    <tr className="border-b border-panelBorder text-foreground">
      <td className="py-3 pr-3 font-medium">
        {pos.perpName || pos.perp.slice(0, 10)}
      </td>
      <td className={`py-3 pr-3 ${isLong ? "text-buy" : "text-sell"}`}>
        {isLong ? "多" : "空"}
      </td>
      <td className="py-3 pr-3 text-right font-mono">
        {formatNumber(Math.abs(size), 4)}
      </td>
      <td className="py-3 pr-3 text-right font-mono">
        {formatNumber(markPriceToUsd(pos.entryPrice), 2)}
      </td>
      <td className="py-3 pr-3 text-right font-mono">
        {formatNumber(markPriceToUsd(pos.markPrice), 2)}
      </td>
      <td className="py-3 pr-3 text-right font-mono text-subtle">
        {formatNumber(notional, 2)}
      </td>
      <td className={`py-3 pr-3 text-right font-mono ${pnlClass(pnl)}`}>
        {pnl >= 0 ? "+" : ""}
        {formatNumber(pnl, 2)}
      </td>
      <td className={`py-3 pr-3 text-right font-mono ${pnlClass(pnlRatio)}`}>
        {pnlRatio >= 0 ? "+" : ""}
        {formatPercent(pnlRatio, 2)}
      </td>
      <td className="py-3 text-right font-mono text-subtle">
        {liq > 0 ? formatNumber(liq, 2) : "—"}
      </td>
    </tr>
  );
}

export default function PositionsPage() {
  const { address, isConnected } = useAccount();
  const [positions, setPositions] = useState<PositionDTO[]>([]);
  const [risk, setRisk] = useState<RiskInfoDTO | null>(null);
  const [loading, setLoading] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const [lastUpdated, setLastUpdated] = useState<number | null>(null);

  const load = useCallback(async () => {
    if (!address || !isConnected) {
      setPositions([]);
      setRisk(null);
      return;
    }
    setLoading(true);
    setErr(null);
    try {
      const trader = address.trim();
      const [p, r] = await Promise.all([
        fetchMetanodePositions(trader),
        fetchMetanodeRisk(trader),
      ]);
      if (p.code !== 0) {
        setErr(p.message || "持仓加载失败");
        setPositions([]);
      } else {
        setPositions(p.positions ?? []);
      }
      if (r.code === 0) {
        setRisk(r.riskInfo);
      }
      setLastUpdated(Date.now());
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, [address, isConnected]);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    if (!isConnected || !address) return;
    const id = window.setInterval(() => void load(), REFRESH_MS);
    return () => window.clearInterval(id);
  }, [address, isConnected, load]);

  const totalPnl = useMemo(() => sumUnrealizedPnl(positions), [positions]);

  return (
    <div className="min-h-screen bg-page text-foreground">
      <header className="flex flex-wrap items-center justify-between gap-3 border-b border-panelBorder px-4 py-3 md:px-6">
        <div className="flex flex-wrap items-center gap-3">
          <Link
            href="/"
            className="rounded-lg bg-elevated px-3 py-2 text-sm font-semibold text-foreground hover:bg-panelBorder"
          >
            返回交易
          </Link>
          <h1 className="text-lg font-semibold text-white">头寸管理</h1>
        </div>
        <ConnectButton />
      </header>

      <main className="mx-auto max-w-5xl space-y-6 px-4 py-6 md:px-6">
        {!isConnected || !address ? (
          <p className="rounded-xl border border-panelBorder bg-panel p-4 text-subtle">
            请先连接钱包，即可查看链上持仓与未实现盈亏。
          </p>
        ) : (
          <>
            <section className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
              <div className="rounded-xl border border-panelBorder bg-panel p-4">
                <div className="text-xs text-subtle">未实现盈亏</div>
                <div className={`mt-1 text-xl font-semibold ${pnlClass(totalPnl)}`}>
                  {totalPnl >= 0 ? "+" : ""}
                  {formatNumber(totalPnl, 2)} USDC
                </div>
                <div className="mt-1 text-[11px] text-subtle">
                  {positions.length} 个持仓
                </div>
              </div>
              {risk ? (
                <>
                  <div className="rounded-xl border border-panelBorder bg-panel p-4">
                    <div className="text-xs text-subtle">账户净值</div>
                    <div className="mt-1 text-xl font-semibold text-foreground">
                      {formatUsdc(risk.netValue)} USDC
                    </div>
                  </div>
                  <div className="rounded-xl border border-panelBorder bg-panel p-4">
                    <div className="text-xs text-subtle">可用保证金</div>
                    <div className="mt-1 text-xl font-semibold text-foreground">
                      {formatUsdc(risk.availableMargin)} USDC
                    </div>
                  </div>
                  <div className="rounded-xl border border-panelBorder bg-panel p-4">
                    <div className="text-xs text-subtle">风险状态</div>
                    <div
                      className={`mt-1 text-xl font-semibold ${risk.isSafe ? "text-buy" : "text-sell"}`}
                    >
                      {risk.isSafe ? "安全" : "需关注"}
                    </div>
                    <div className="mt-1 text-[11px] text-subtle">
                      维持保证金 {formatUsdc(risk.maintenanceMargin)} USDC
                    </div>
                  </div>
                </>
              ) : null}
            </section>

            <section className="rounded-xl border border-panelBorder bg-panel p-4">
              <div className="mb-4 flex flex-wrap items-center justify-between gap-2">
                <h2 className="text-sm font-semibold text-white">持仓明细</h2>
                <div className="flex items-center gap-2 text-[11px] text-subtle">
                  {lastUpdated ? (
                    <span>
                      更新于{" "}
                      {new Date(lastUpdated).toLocaleTimeString("zh-CN")}
                    </span>
                  ) : null}
                  <button
                    type="button"
                    disabled={loading}
                    onClick={() => void load()}
                    className="rounded-md border border-panelBorder px-2 py-1 text-subtle hover:bg-panelBorder disabled:opacity-50"
                  >
                    {loading ? "刷新中…" : "刷新"}
                  </button>
                </div>
              </div>

              {err ? (
                <p className="py-6 text-center text-red-400">{err}</p>
              ) : positions.length === 0 ? (
                <p className="py-8 text-center text-subtle">
                  {loading
                    ? "加载中…"
                    : "暂无持仓。下单并撮合成交后，仓位会出现在这里。"}
                </p>
              ) : (
                <div className="overflow-x-auto">
                  <table className="min-w-[880px] w-full text-left text-xs">
                    <thead>
                      <tr className="border-b border-panelBorder text-subtle">
                        <th className="pb-2 pr-3 font-medium">市场</th>
                        <th className="pb-2 pr-3 font-medium">方向</th>
                        <th className="pb-2 pr-3 text-right font-medium">数量</th>
                        <th className="pb-2 pr-3 text-right font-medium">
                          开仓价
                        </th>
                        <th className="pb-2 pr-3 text-right font-medium">
                          标记价
                        </th>
                        <th className="pb-2 pr-3 text-right font-medium">
                          名义价值
                        </th>
                        <th className="pb-2 pr-3 text-right font-medium">
                          未实现盈亏
                        </th>
                        <th className="pb-2 pr-3 text-right font-medium">
                          收益率
                        </th>
                        <th className="pb-2 text-right font-medium">
                          强平价
                        </th>
                      </tr>
                    </thead>
                    <tbody>
                      {positions.map((pos) => (
                        <PositionRow
                          key={`${pos.perp}-${pos.paper}`}
                          pos={pos}
                        />
                      ))}
                    </tbody>
                  </table>
                </div>
              )}

              <p className="mt-4 text-[11px] text-subtle">
                盈亏按标记价相对开仓价估算；平仓请返回
                <Link href="/" className="mx-1 text-accent hover:underline">
                  交易页
                </Link>
                操作。
              </p>
            </section>
          </>
        )}
      </main>
    </div>
  );
}
