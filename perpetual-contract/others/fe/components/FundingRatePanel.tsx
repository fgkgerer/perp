"use client";

import { useEffect, useMemo, useState } from "react";
import {
  fetchMetanodeFundingLatest,
  formatCountdown,
  formatFundingRatePct,
  fundingRateTone,
  nextFundingSettleMsClient,
  type FundingRateLatestDTO,
} from "@/lib/funding";

type Props = {
  perp: string;
  /** 紧凑模式用于顶栏等窄区域 */
  compact?: boolean;
};

export default function FundingRatePanel({ perp, compact = false }: Props) {
  const [latest, setLatest] = useState<FundingRateLatestDTO | null>(null);
  const [loadErr, setLoadErr] = useState<string | null>(null);
  // 服务端与客户端首帧保持一致，挂载后再启动倒计时，避免 hydration 抖动。
  const [now, setNow] = useState<number | null>(null);

  useEffect(() => {
    if (!perp) return;
    let cancelled = false;
    setLoadErr(null);
    void (async () => {
      try {
        const resp = await fetchMetanodeFundingLatest(perp);
        if (cancelled) return;
        if (resp.code === 0 && resp.latest) {
          setLatest(resp.latest);
        } else {
          setLatest(null);
          setLoadErr(resp.message || "暂无资金费数据");
        }
      } catch (e) {
        if (!cancelled) {
          setLatest(null);
          setLoadErr(e instanceof Error ? e.message : String(e));
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [perp]);

  useEffect(() => {
    setNow(Date.now());
    const id = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(id);
  }, []);

  const nextMs = useMemo(() => {
    if (latest?.nextSettleAt && latest.nextSettleAt > 0) {
      return latest.nextSettleAt * 1000;
    }
    return now == null ? 0 : nextFundingSettleMsClient(now);
  }, [latest?.nextSettleAt, now]);

  const countdown = now == null ? "--:--:--" : formatCountdown(nextMs - now);

  const periodPct =
    latest?.periodRateSource === "last_settle"
      ? latest.periodRatePct
      : latest?.predictedPeriodRatePct ?? latest?.periodRatePct ?? "";
  const tone = fundingRateTone(periodPct);
  const rateClass =
    tone === "positive"
      ? "text-buy"
      : tone === "negative"
        ? "text-sell"
        : "text-foreground";

  if (compact) {
    return (
      <div className="flex flex-col items-end text-right text-xs">
        <div className="text-subtle">资金费率 / 8h</div>
        <div className={`font-mono font-semibold ${rateClass}`}>
          {periodPct ? formatFundingRatePct(periodPct) : "—"}
        </div>
        <div className="mt-0.5 font-mono text-[11px] text-muted">
          <span className="text-subtle">结算 </span>
          {countdown}
        </div>
        {loadErr ? (
          <div className="mt-1 max-w-[200px] text-[10px] text-amber-400/90">{loadErr}</div>
        ) : null}
      </div>
    );
  }

  return (
    <div className="min-w-[140px]">
      <div className="text-subtle">资金费率（8h）</div>
      <div className={`mt-0.5 font-mono text-base font-semibold ${rateClass}`}>
        {periodPct ? formatFundingRatePct(periodPct) : "—"}
      </div>
      {latest?.periodRateSource === "predicted" && periodPct ? (
        <div className="mt-0.5 text-[10px] text-faint">预估（mark − index）</div>
      ) : latest?.periodRateSource === "last_settle" ? (
        <div className="mt-0.5 text-[10px] text-faint">上期结算</div>
      ) : null}
      <div className="mt-2 text-subtle">距下次结算</div>
      <div className="font-mono text-lg font-medium tracking-tight text-accent">
        {countdown}
      </div>
      <div className="mt-1 text-[10px] leading-snug text-faint">
        {latest?.settleSchedule ?? "UTC 每 8 小时：00:00、08:00、16:00"}
      </div>
      {loadErr ? (
        <div className="mt-1 text-[10px] text-amber-400/90">{loadErr}</div>
      ) : null}
    </div>
  );
}
