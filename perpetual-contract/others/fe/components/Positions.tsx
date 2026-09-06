"use client";

import { useCallback, useEffect, useState } from "react";
import { formatUnits } from "viem";
import { useAccount } from "wagmi";
import {
  fetchMetanodePositions,
  fetchMetanodeRisk,
  type PositionDTO,
  type RiskInfoDTO,
} from "@/lib/metanode-api";
import {
  markPriceToUsd,
  paperToSize,
} from "@/lib/metanode-markets";
import { calcPnlRatio, calcUnrealizedPnl } from "@/lib/position-pnl";
import { buildCloseOrderInput } from "@/lib/metanode-order";
import { formatNumber, formatPercent } from "@/lib/format";
import type { CloseDraft } from "@/components/OrderForm";

type Props = {
  refreshKey?: number;
  onRequestClose?: (draft: CloseDraft) => void;
  embedded?: boolean;
};

const USDC_DECIMALS = 6;

function formatUsdc(raw: string): string {
  try {
    return formatUnits(BigInt(raw.trim() || "0"), USDC_DECIMALS);
  } catch {
    return raw;
  }
}

export default function Positions({
  refreshKey = 0,
  onRequestClose,
  embedded = false,
}: Props) {
  const { address, isConnected } = useAccount();
  const [positions, setPositions] = useState<PositionDTO[]>([]);
  const [risk, setRisk] = useState<RiskInfoDTO | null>(null);
  const [loading, setLoading] = useState(false);
  const [err, setErr] = useState<string | null>(null);

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
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, [address, isConnected]);

  useEffect(() => {
    void load();
  }, [load, refreshKey]);

  const handleClose = (pos: PositionDTO) => {
    if (!address) return;
    try {
      const priceUsd = String(
        Math.round(markPriceToUsd(pos.markPrice) || markPriceToUsd(pos.entryPrice))
      );
      const input = buildCloseOrderInput(
        pos.perp as `0x${string}`,
        address,
        pos.paper,
        priceUsd
      );
      onRequestClose?.({
        perp: input.perp,
        size: input.size,
        priceUsd: input.priceUsd,
        side: input.side,
      });
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    }
  };

  const inner = (
    <>
      <div
        className={`flex items-center justify-between gap-2 ${embedded ? "px-2 pt-1" : "px-4 pt-3"}`}
      >
        {!embedded ? (
          <div className="panel-header border-0 p-0">MetaNode 持仓</div>
        ) : (
          <span className="text-[11px] text-subtle">链上仓位</span>
        )}
        <button
          type="button"
          disabled={loading || !isConnected}
          onClick={() => void load()}
          className="rounded-md border border-panelBorder px-2 py-1 text-[11px] text-subtle hover:bg-white/5 disabled:opacity-50"
        >
          {loading ? "刷新中…" : "刷新"}
        </button>
      </div>

      {risk ? (
        <div className="mx-4 mb-2 grid grid-cols-2 gap-2 rounded-lg border border-panelBorder bg-elevated px-3 py-2 text-[11px] text-subtle sm:grid-cols-4">
          <div>
            <div>净值</div>
            <div className="font-mono text-foreground">
              {formatUsdc(risk.netValue)}
            </div>
          </div>
          <div>
            <div>维持保证金</div>
            <div className="font-mono text-foreground">
              {formatUsdc(risk.maintenanceMargin)}
            </div>
          </div>
          <div>
            <div>可用</div>
            <div className="font-mono text-foreground">
              {formatUsdc(risk.availableMargin)}
            </div>
          </div>
          <div>
            <div>安全</div>
            <div className={risk.isSafe ? "text-buy" : "text-sell"}>
              {risk.isSafe ? "是" : "否"}
            </div>
          </div>
        </div>
      ) : null}

      <div className="overflow-x-auto px-4 pb-4 pt-2 text-xs text-muted">
        {!isConnected ? (
          <p className="py-4 text-subtle">连接钱包后查看链上持仓</p>
        ) : err ? (
          <p className="py-4 text-red-400">{err}</p>
        ) : positions.length === 0 ? (
          <p className="py-4 text-subtle">
            {loading ? "加载中…" : "暂无持仓（需撮合成交后才有仓位）"}
          </p>
        ) : (
          <>
            <div className="grid min-w-[860px] grid-cols-8 text-[11px] text-subtle">
              <span>市场</span>
              <span>方向</span>
              <span className="text-right">数量</span>
              <span className="text-right">开仓价</span>
              <span className="text-right">标记价</span>
              <span className="text-right">盈亏</span>
              <span className="text-right">收益率</span>
              <span className="text-right">操作</span>
            </div>
            <div className="mt-3 space-y-2">
              {positions.map((pos) => {
                const size = paperToSize(pos.paper);
                const isLong = size > 0;
                const pnl = calcUnrealizedPnl(pos);
                const pnlRatio = calcPnlRatio(pos);
                const pnlClass =
                  pnl > 0 ? "text-buy" : pnl < 0 ? "text-sell" : "text-subtle";
                return (
                  <div
                    key={`${pos.perp}-${pos.paper}`}
                    className="grid min-w-[860px] grid-cols-8 items-center text-foreground"
                  >
                    <span>{pos.perpName || pos.perp.slice(0, 8)}</span>
                    <span className={isLong ? "text-buy" : "text-sell"}>
                      {isLong ? "多" : "空"}
                    </span>
                    <span className="text-right">
                      {formatNumber(Math.abs(size), 4)}
                    </span>
                    <span className="text-right">
                      {formatNumber(markPriceToUsd(pos.entryPrice), 2)}
                    </span>
                    <span className="text-right">
                      {formatNumber(markPriceToUsd(pos.markPrice), 2)}
                    </span>
                    <span className={`text-right ${pnlClass}`}>
                      {pnl >= 0 ? "+" : ""}
                      {formatNumber(pnl, 2)}
                    </span>
                    <span className={`text-right ${pnlClass}`}>
                      {pnlRatio >= 0 ? "+" : ""}
                      {formatPercent(pnlRatio, 2)}
                    </span>
                    <span className="text-right">
                      <button
                        type="button"
                        onClick={() => handleClose(pos)}
                        className="rounded border border-panelBorder px-2 py-1 text-[11px] hover:bg-white/5"
                      >
                        平仓
                      </button>
                    </span>
                  </div>
                );
              })}
            </div>
          </>
        )}
      </div>
    </>
  );

  if (embedded) return inner;

  return <div className="panel">{inner}</div>;
}
