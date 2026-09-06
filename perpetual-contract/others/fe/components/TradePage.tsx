"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { KlinePoint } from "@/lib/types";
import Chart from "@/components/Chart";
import OrderForm, { type CloseDraft } from "@/components/OrderForm";
import Positions from "@/components/Positions";
import MetaNodeTopBar from "@/components/MetaNodeTopBar";
import FuturesContractBar from "@/components/FuturesContractBar";
import MarketDepthTabs from "@/components/MarketDepthTabs";
import { orderBookBidShare } from "@/components/OrderBook";
import Trades, { type DisplayTrade } from "@/components/Trades";
import {
  fetchMetanodeKlines,
  fetchMetanodeMarkets,
  fetchMetanodeOrderBook,
  fetchMetanodeTrades,
  METANODE_API_BASE,
  type OrderBookEntryDTO,
  type PerpMarketDTO,
} from "@/lib/metanode-api";
import {
  quotePriceUsd,
  usdToIndexPrice1e6,
  type MarketQuoteRow,
} from "@/lib/market-quote";
import { useMarketQuotesRealtime } from "@/lib/useMarketQuotesRealtime";
import { isSupabaseConfigured } from "@/lib/supabase";
import {
  markPriceToUsd,
  paperToSize,
  SEPOLIA_METANODE_MARKETS,
  type MetaNodeMarket,
} from "@/lib/metanode-markets";

const RESOLUTIONS = [
  { label: "1分", value: "1m" },
  { label: "5分", value: "5m" },
  { label: "15分", value: "15" },
  { label: "1小时", value: "60" },
  { label: "4小时", value: "240" },
  { label: "1天", value: "1D" },
];

const POLL_MS = 5_000;
const CHAIN_MARKETS_POLL_MS = 30_000;

function intervalSeconds(resolution: string): number {
  if (resolution === "1m") return 60;
  if (resolution === "5m") return 5 * 60;
  if (resolution === "60") return 60 * 60;
  if (resolution === "240") return 4 * 60 * 60;
  if (resolution === "1D") return 24 * 60 * 60;
  return 15 * 60;
}

function buildMarkKline(
  markUsd: number,
  resolution: string,
  points = 120
): KlinePoint[] {
  if (markUsd <= 0) return [];
  const interval = intervalSeconds(resolution);
  const now = Math.floor(Date.now() / 1000);
  const aligned = Math.floor(now / interval) * interval;
  const out: KlinePoint[] = [];
  for (let i = points - 1; i >= 0; i--) {
    const t = aligned - i * interval;
    out.push({
      time: t,
      open: markUsd,
      high: markUsd,
      low: markUsd,
      close: markUsd,
      volume: 0,
    });
  }
  return out;
}

function patchKlineWithMark(
  prev: KlinePoint[],
  markUsd: number,
  resolution: string
): KlinePoint[] {
  if (markUsd <= 0) return prev;
  const interval = intervalSeconds(resolution);
  const now = Math.floor(Date.now() / 1000);
  const aligned = Math.floor(now / interval) * interval;

  if (prev.length === 0) {
    return buildMarkKline(markUsd, resolution);
  }

  const next = [...prev];
  const lastIdx = next.length - 1;
  const last = next[lastIdx];

  if (aligned > last.time) {
    next.push({
      time: aligned,
      open: markUsd,
      high: markUsd,
      low: markUsd,
      close: markUsd,
      volume: 0,
    });
    if (next.length > 200) next.shift();
    return next;
  }

  next[lastIdx] = {
    ...last,
    close: markUsd,
    high: Math.max(last.high, markUsd),
    low: Math.min(last.low, markUsd),
  };
  return next;
}

function tradesToDisplay(
  rows: Awaited<ReturnType<typeof fetchMetanodeTrades>>["trades"]
): DisplayTrade[] {
  return rows.map((t) => ({
    price: markPriceToUsd(t.price),
    size: Math.abs(paperToSize(t.paperAmount)),
    ts: t.createTime,
    side: "BUY" as const,
    txHash: t.txHash,
  }));
}

export default function TradePage({
  onOpenProfile,
}: {
  onOpenProfile?: () => void;
}) {
  const [resolution, setResolution] = useState("15");
  const [chainMarkets, setChainMarkets] = useState<PerpMarketDTO[]>([]);
  const [metaMarkets, setMetaMarkets] = useState<MetaNodeMarket[]>(
    SEPOLIA_METANODE_MARKETS
  );
  const [selectedPerp, setSelectedPerp] = useState<`0x${string}`>(
    SEPOLIA_METANODE_MARKETS[0].address
  );
  const [markPrices, setMarkPrices] = useState<Record<string, string>>({});
  const [indexPricesUsd, setIndexPricesUsd] = useState<Record<string, string>>({});
  const [kline, setKline] = useState<KlinePoint[]>([]);
  const [klineLoading, setKlineLoading] = useState(false);
  const [bookBids, setBookBids] = useState<OrderBookEntryDTO[]>([]);
  const [bookAsks, setBookAsks] = useState<OrderBookEntryDTO[]>([]);
  const [bookLoading, setBookLoading] = useState(false);
  const [recentTrades, setRecentTrades] = useState<DisplayTrade[]>([]);
  const [tradesLoading, setTradesLoading] = useState(false);
  const [dataError, setDataError] = useState<string | null>(null);
  const [closeDraft, setCloseDraft] = useState<CloseDraft | null>(null);
  const [positionsRefresh, setPositionsRefresh] = useState(0);

  const currentMarkUsd = useMemo(
    () => markPriceToUsd(markPrices[selectedPerp.toLowerCase()]),
    [markPrices, selectedPerp]
  );

  const currentIndexUsd = useMemo(() => {
    const idx = indexPricesUsd[selectedPerp.toLowerCase()];
    if (idx) return quotePriceUsd(idx);
    return currentMarkUsd;
  }, [indexPricesUsd, selectedPerp, currentMarkUsd]);

  const applyQuote = useCallback((q: MarketQuoteRow) => {
    const perp = q.perp.toLowerCase();
    setIndexPricesUsd((prev) => ({ ...prev, [perp]: q.price_usd }));
  }, []);

  const { status: realtimeStatus, lastError: realtimeError } =
    useMarketQuotesRealtime({ onQuote: applyQuote });

  const displayMarkets = useMemo(() => {
    const base =
      chainMarkets.length > 0 ? chainMarkets : chainMarketsFromFallback();
    return base.map((m) => {
      const liveUsd = indexPricesUsd[m.address.toLowerCase()];
      if (!liveUsd) return m;
      return { ...m, indexPrice: usdToIndexPrice1e6(liveUsd) };
    });
  }, [chainMarkets, indexPricesUsd]);

  const selectedMarketDto = useMemo(
    () =>
      displayMarkets.find(
        (m) => m.address.toLowerCase() === selectedPerp.toLowerCase()
      ),
    [displayMarkets, selectedPerp]
  );

  const bidSharePct = useMemo(
    () => orderBookBidShare(bookBids, bookAsks),
    [bookBids, bookAsks]
  );

  const dayRange = useMemo(() => {
    const cutoff = Math.floor(Date.now() / 1000) - 24 * 60 * 60;
    const points = kline.filter(
      (point) =>
        point.time >= cutoff &&
        Number.isFinite(point.low) &&
        Number.isFinite(point.high)
    );
    if (points.length === 0) return { low: undefined, high: undefined };
    return {
      low: Math.min(...points.map((point) => point.low)),
      high: Math.max(...points.map((point) => point.high)),
    };
  }, [kline]);

  const symbol =
    metaMarkets.find((m) => m.address.toLowerCase() === selectedPerp.toLowerCase())
      ?.symbol ?? "BTC";

  const loadMarkets = useCallback(async () => {
    try {
      const resp = await fetchMetanodeMarkets();
      if (resp.code !== 0 || !resp.markets?.length) return;
      setChainMarkets(
        resp.markets.map((m) => {
          const live = indexPricesUsd[m.address.toLowerCase()];
          return {
            ...m,
            indexPrice: live ? usdToIndexPrice1e6(live) : m.indexPrice,
          };
        })
      );
      const mapped: MetaNodeMarket[] = resp.markets.map((m) => ({
        name: m.name,
        address: m.address as `0x${string}`,
        symbol: m.name.split("-")[0] ?? m.name,
      }));
      setMetaMarkets(mapped);
      const prices: Record<string, string> = {};
      for (const m of resp.markets) {
        if (m.markPrice) prices[m.address.toLowerCase()] = m.markPrice;
      }
      setMarkPrices(prices);
      setSelectedPerp((prev) => {
        if (mapped.some((m) => m.address === prev)) return prev;
        return mapped[0].address;
      });
      setDataError(null);
    } catch (e) {
      setDataError(e instanceof Error ? e.message : String(e));
    }
  }, [indexPricesUsd]);

  const loadOrderBook = useCallback(async () => {
    setBookLoading(true);
    try {
      const resp = await fetchMetanodeOrderBook(selectedPerp, 20);
      if (resp.code !== 0) {
        setBookBids([]);
        setBookAsks([]);
        return;
      }
      setBookBids(resp.bids ?? []);
      setBookAsks(resp.asks ?? []);
    } catch {
      setBookBids([]);
      setBookAsks([]);
    } finally {
      setBookLoading(false);
    }
  }, [selectedPerp]);

  const loadKlines = useCallback(async () => {
    setKlineLoading(true);
    try {
      const resp = await fetchMetanodeKlines(selectedPerp, resolution, {
        limit: 200,
      });
      if (resp.code !== 0 || !resp.klines?.length) {
        const mark = markPriceToUsd(markPrices[selectedPerp.toLowerCase()]);
        setKline(buildMarkKline(mark, resolution));
        return;
      }
      const points: KlinePoint[] = resp.klines.map((k) => ({
        time: k.time,
        open: Number.parseFloat(k.open),
        high: Number.parseFloat(k.high),
        low: Number.parseFloat(k.low),
        close: Number.parseFloat(k.close),
        volume: Number.parseFloat(k.volume) || 0,
      }));
      setKline(points.filter((p) => Number.isFinite(p.close) && p.close > 0));
    } catch {
      const mark = markPriceToUsd(markPrices[selectedPerp.toLowerCase()]);
      setKline(buildMarkKline(mark, resolution));
    } finally {
      setKlineLoading(false);
    }
  }, [selectedPerp, resolution, markPrices]);

  const loadTrades = useCallback(async () => {
    setTradesLoading(true);
    try {
      const resp = await fetchMetanodeTrades(selectedPerp, 50);
      if (resp.code !== 0) {
        setRecentTrades([]);
        return;
      }
      setRecentTrades(tradesToDisplay(resp.trades ?? []));
    } catch {
      setRecentTrades([]);
    } finally {
      setTradesLoading(false);
    }
  }, [selectedPerp]);

  useEffect(() => {
    void loadMarkets();
    const id = window.setInterval(() => void loadMarkets(), CHAIN_MARKETS_POLL_MS);
    return () => window.clearInterval(id);
  }, [loadMarkets]);

  useEffect(() => {
    void loadOrderBook();
    void loadTrades();
    const id = window.setInterval(() => {
      void loadOrderBook();
      void loadTrades();
    }, POLL_MS);
    return () => window.clearInterval(id);
  }, [loadOrderBook, loadTrades]);

  useEffect(() => {
    void loadKlines();
  }, [loadKlines]);

  useEffect(() => {
    if (currentIndexUsd <= 0) return;
    setKline((prev) => patchKlineWithMark(prev, currentIndexUsd, resolution));
  }, [currentIndexUsd, resolution]);

  const refreshAfterOrder = () => {
    setPositionsRefresh((k) => k + 1);
    void loadOrderBook();
    void loadTrades();
  };

  const quoteStatus = dataError
    ? "链上行情连接异常"
    : !isSupabaseConfigured()
      ? "指数行情 HTTP 轮询"
      : realtimeStatus === "error"
        ? "实时行情重连中"
        : realtimeStatus === "subscribed"
          ? "行情实时同步"
          : "行情连接中";
  const quoteStatusError = Boolean(dataError || realtimeStatus === "error");

  return (
    <div className="min-h-screen bg-page pb-6 pt-20">
      <MetaNodeTopBar
        markets={displayMarkets}
        selectedPerp={selectedPerp}
        onSelectPerp={(addr) => setSelectedPerp(addr as `0x${string}`)}
        onOpenProfile={onOpenProfile}
        indexPriceUsd={currentIndexUsd}
        realtimeStatus={realtimeStatus}
        compact
      />

      <FuturesContractBar
        market={selectedMarketDto}
        markets={displayMarkets}
        selectedPerp={selectedPerp}
        onSelectPerp={(addr) => setSelectedPerp(addr as `0x${string}`)}
        indexPriceUsd={currentIndexUsd}
        bidSharePct={bidSharePct}
      />

      {/* 专业交易终端：K 线 | 订单簿/成交 | 下单 */}
      <main className="mx-auto max-w-[1920px] p-1.5 sm:p-2">
        <div className="grid grid-cols-1 gap-px overflow-hidden border border-panelBorder bg-panelBorder xl:h-[calc(100vh-192px)] xl:min-h-[540px] xl:grid-cols-[minmax(520px,1fr)_300px_340px] 2xl:grid-cols-[minmax(640px,1fr)_320px_360px]">
          <section className="min-h-[420px] min-w-0 bg-panel xl:min-h-0">
            <Chart
              data={kline}
              loading={klineLoading}
              error={currentIndexUsd <= 0 ? "等待指数价…" : null}
              resolution={resolution}
              onResolutionChange={setResolution}
              options={RESOLUTIONS}
              symbol={symbol}
              depthBids={bookBids.map((entry) => ({
                price: markPriceToUsd(entry.price),
                amount: Math.abs(paperToSize(entry.amount)),
              }))}
              depthAsks={bookAsks.map((entry) => ({
                price: markPriceToUsd(entry.price),
                amount: Math.abs(paperToSize(entry.amount)),
              }))}
              trades={recentTrades}
            />
          </section>

          <section className="flex min-h-[420px] min-w-0 flex-col bg-panel xl:min-h-0">
            <MarketDepthTabs
              bids={bookBids}
              asks={bookAsks}
              trades={recentTrades}
              bookLoading={bookLoading}
              tradesLoading={tradesLoading}
              markPriceUsd={currentMarkUsd}
              symbol={symbol}
              className="min-h-[420px] flex-1 border-0 xl:min-h-0"
            />
          </section>

          <aside className="terminal-scrollbar min-w-0 bg-panel xl:h-full xl:overflow-y-auto">
            <OrderForm
              markets={metaMarkets}
              selectedPerp={selectedPerp}
              onSelectPerp={setSelectedPerp}
              marketRisk={selectedMarketDto}
              markPriceRaw={markPrices[selectedPerp.toLowerCase()]}
              indexPriceUsd={currentIndexUsd}
              bidSharePct={bidSharePct}
              dayLowUsd={dayRange.low}
              dayHighUsd={dayRange.high}
              hideMarketSelect
              closeDraft={closeDraft}
              onCloseDraftApplied={() => setCloseDraft(null)}
              onOrderSubmitted={refreshAfterOrder}
            />
          </aside>
        </div>

        <div className="mt-2 overflow-hidden border border-panelBorder bg-panel">
          <div className="flex h-10 items-center justify-between border-b border-panelBorder px-4">
            <div className="flex h-full items-center gap-6 text-xs">
              <span className="relative flex h-full items-center font-semibold text-white after:absolute after:bottom-0 after:left-0 after:right-0 after:h-0.5 after:bg-accent">当前仓位</span>
              <span className="text-subtle">当前委托</span>
              <span className="text-subtle">历史成交</span>
            </div>
            <span className="hidden text-[10px] text-subtle sm:block">数据来自 Sepolia · 自动更新</span>
          </div>
          <div className="p-2">
            <Positions
              refreshKey={positionsRefresh}
              onRequestClose={(draft) => setCloseDraft(draft)}
              embedded
            />
          </div>
        </div>
      </main>

      <div className="fixed bottom-0 left-0 right-0 z-40 flex h-6 items-center justify-between border-t border-panelBorder bg-[#070a0b] px-3 text-[9px] text-subtle">
        <div
          className="flex min-w-0 items-center gap-2"
          title={dataError ?? realtimeError ?? undefined}
        >
          <span className={`h-1.5 w-1.5 shrink-0 rounded-full ${quoteStatusError ? "bg-amber-400" : "bg-buy"}`} />
          <span className={quoteStatusError ? "text-amber-300" : ""}>{quoteStatus}</span>
          {quoteStatusError ? <span className="hidden truncate text-faint sm:block">· 请检查 {METANODE_API_BASE}</span> : null}
        </div>
        <div className="flex items-center gap-4">
          <span className="hidden sm:inline">UTC+8</span>
          <span className="text-muted">Sepolia 永续合约</span>
        </div>
      </div>
    </div>
  );
}

function chainMarketsFromFallback(): PerpMarketDTO[] {
  return SEPOLIA_METANODE_MARKETS.map((m) => {
    const key = m.address.toLowerCase();
    const risk =
      key === "0x11aae1f92ff10bfbb205971e060cf6d9d917723b"
        ? {
            maxLeverage: "20",
            initialMarginPct: "5.00",
            maintenanceMarginPct: "3.00",
          }
        : {
            maxLeverage: "10",
            initialMarginPct: "10.00",
            maintenanceMarginPct: "5.00",
          };
    return {
      address: m.address,
      name: m.name,
      markPrice: "0",
      indexPrice: "0",
      fundingRate: "0",
      isRegistered: true,
      ...risk,
    };
  });
}
