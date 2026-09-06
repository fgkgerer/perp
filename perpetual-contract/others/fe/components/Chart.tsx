"use client";

import { useEffect, useMemo, useRef, useState, type ReactElement } from "react";
import {
  ArrowsPointingOutIcon,
  CameraIcon,
  ChartBarIcon,
  ChevronDownIcon,
  Cog6ToothIcon,
  CursorArrowRaysIcon,
  MagnifyingGlassPlusIcon,
  MinusIcon,
  PencilSquareIcon,
  PresentationChartLineIcon,
  Squares2X2Icon,
} from "@heroicons/react/24/outline";
import {
  CandlestickSeries,
  ColorType,
  CrosshairMode,
  HistogramSeries,
  ISeriesApi,
  LineSeries,
  UTCTimestamp,
  createChart,
} from "lightweight-charts";
import { KlinePoint } from "@/lib/types";
import { formatNumber, formatPercent, formatTime } from "@/lib/format";

type ChartView = "kline" | "info" | "depth" | "trades";

type Option = {
  label: string;
  value: string;
};

type DepthLevel = {
  price: number;
  amount: number;
};

type ChartTrade = {
  price: number;
  size: number;
  ts: number;
  side: "BUY" | "SELL";
};

type Props = {
  data: KlinePoint[];
  loading: boolean;
  error: string | null;
  resolution: string;
  onResolutionChange: (value: string) => void;
  options: Option[];
  symbol?: string;
  depthBids?: DepthLevel[];
  depthAsks?: DepthLevel[];
  trades?: ChartTrade[];
};

const VIEW_TABS: Array<{ id: ChartView; label: string }> = [
  { id: "kline", label: "K线" },
  { id: "info", label: "信息" },
  { id: "depth", label: "深度" },
  { id: "trades", label: "交易数据" },
];

function movingAverage(data: KlinePoint[], period: number) {
  const result: Array<{ time: UTCTimestamp; value: number }> = [];
  let sum = 0;
  for (let index = 0; index < data.length; index += 1) {
    sum += data[index].close;
    if (index >= period) sum -= data[index - period].close;
    if (index >= period - 1) {
      result.push({
        time: data[index].time as UTCTimestamp,
        value: sum / period,
      });
    }
  }
  return result;
}

function lastValue<T>(rows: T[]): T | undefined {
  return rows[rows.length - 1];
}

export default function Chart({
  data,
  loading,
  error,
  resolution,
  onResolutionChange,
  options,
  symbol = "BTC",
  depthBids = [],
  depthAsks = [],
  trades = [],
}: Props) {
  const containerRef = useRef<HTMLDivElement>(null);
  const chartRef = useRef<ReturnType<typeof createChart> | null>(null);
  const seriesRef = useRef<ISeriesApi<"Candlestick"> | null>(null);
  const volumeSeriesRef = useRef<ISeriesApi<"Histogram"> | null>(null);
  const ma5Ref = useRef<ISeriesApi<"Line"> | null>(null);
  const ma10Ref = useRef<ISeriesApi<"Line"> | null>(null);
  const ma30Ref = useRef<ISeriesApi<"Line"> | null>(null);
  const fittedResolutionRef = useRef<string | null>(null);
  const [view, setView] = useState<ChartView>("kline");
  const [hoverData, setHoverData] = useState<{
    open: number;
    high: number;
    low: number;
    close: number;
    change: number;
  } | null>(null);

  const latestData = useMemo(() => {
    if (!data.length) return null;
    const last = data[data.length - 1];
    const prev = data.length > 1 ? data[data.length - 2] : last;
    const change = prev.close ? (last.close - prev.close) / prev.close : 0;
    return { ...last, change };
  }, [data]);

  const maValues = useMemo(() => {
    const ma5 = movingAverage(data, 5);
    const ma10 = movingAverage(data, 10);
    const ma30 = movingAverage(data, 30);
    return {
      ma5,
      ma10,
      ma30,
      latest5: lastValue(ma5)?.value,
      latest10: lastValue(ma10)?.value,
      latest30: lastValue(ma30)?.value,
    };
  }, [data]);

  const displayData = hoverData || latestData;

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    const chart = createChart(container, {
      layout: {
        background: { type: ColorType.Solid, color: "#070a0b" },
        textColor: "#778184",
        fontFamily: "ui-sans-serif, system-ui",
        fontSize: 11,
      },
      grid: {
        vertLines: { color: "#182023", style: 0 },
        horzLines: { color: "#182023", style: 0 },
      },
      rightPriceScale: {
        borderColor: "#1d272b",
        scaleMargins: { top: 0.08, bottom: 0.24 },
      },
      timeScale: {
        borderColor: "#1d272b",
        timeVisible: true,
        secondsVisible: false,
        rightOffset: 8,
        barSpacing: 8,
      },
      crosshair: {
        mode: CrosshairMode.Normal,
        vertLine: {
          color: "#596367",
          width: 1,
          style: 3,
          labelBackgroundColor: "#242b2f",
        },
        horzLine: {
          color: "#596367",
          width: 1,
          style: 3,
          labelBackgroundColor: "#242b2f",
        },
      },
    });
    chartRef.current = chart;

    const series = chart.addSeries(CandlestickSeries, {
      upColor: "#13c991",
      downColor: "#f0446e",
      borderUpColor: "#13c991",
      borderDownColor: "#f0446e",
      wickUpColor: "#13c991",
      wickDownColor: "#f0446e",
      priceScaleId: "right",
    });
    seriesRef.current = series;

    const addMa = (color: string) =>
      chart.addSeries(LineSeries, {
        color,
        lineWidth: 2,
        priceLineVisible: false,
        lastValueVisible: false,
        crosshairMarkerVisible: false,
      });
    ma5Ref.current = addMa("#f3a11b");
    ma10Ref.current = addMa("#15b9df");
    ma30Ref.current = addMa("#cd63bd");

    const volumeSeries = chart.addSeries(HistogramSeries, {
      color: "#13c991",
      priceFormat: { type: "volume" },
      priceScaleId: "",
      priceLineVisible: false,
      lastValueVisible: false,
    });
    volumeSeries.priceScale().applyOptions({
      scaleMargins: { top: 0.82, bottom: 0 },
    });
    volumeSeriesRef.current = volumeSeries;

    chart.subscribeCrosshairMove((param) => {
      if (param.time && param.seriesData.has(series) && param.point) {
        const candle = param.seriesData.get(series) as
          | { open: number; high: number; low: number; close: number }
          | undefined;
        if (candle) {
          const change = candle.open
            ? (candle.close - candle.open) / candle.open
            : 0;
          setHoverData({ ...candle, change });
        }
      } else {
        setHoverData(null);
      }
    });

    const resize = () => {
      chart.applyOptions({
        width: container.clientWidth,
        height: container.clientHeight,
      });
    };
    resize();
    const observer = new ResizeObserver(resize);
    observer.observe(container);

    return () => {
      observer.disconnect();
      chart.remove();
      chartRef.current = null;
      seriesRef.current = null;
      volumeSeriesRef.current = null;
      ma5Ref.current = null;
      ma10Ref.current = null;
      ma30Ref.current = null;
    };
  }, []);

  useEffect(() => {
    if (!seriesRef.current) return;
    seriesRef.current.setData(
      data.map((item) => ({
        time: item.time as UTCTimestamp,
        open: item.open,
        high: item.high,
        low: item.low,
        close: item.close,
      }))
    );
    ma5Ref.current?.setData(maValues.ma5);
    ma10Ref.current?.setData(maValues.ma10);
    ma30Ref.current?.setData(maValues.ma30);
    volumeSeriesRef.current?.setData(
      data.map((item) => ({
        time: item.time as UTCTimestamp,
        value: item.volume,
        color: item.close >= item.open ? "#13c99188" : "#f0446e88",
      }))
    );
    if (data.length > 0 && fittedResolutionRef.current !== resolution) {
      chartRef.current?.timeScale().fitContent();
      fittedResolutionRef.current = resolution;
    }
  }, [data, maValues, resolution]);

  return (
    <div className="flex h-full min-h-[420px] flex-col bg-[#070a0b] xl:min-h-0">
      <div
        role="tablist"
        aria-label="图表视图"
        className="flex h-11 shrink-0 items-center gap-1 border-b border-panelBorder px-3"
      >
        {VIEW_TABS.map((tab) => (
          <button
            key={tab.id}
            type="button"
            role="tab"
            aria-selected={view === tab.id}
            aria-controls={`chart-panel-${tab.id}`}
            onClick={() => setView(tab.id)}
            className={`relative h-full border-0 bg-transparent px-3 text-sm font-semibold transition ${
              view === tab.id
                ? "text-white"
                : "text-subtle hover:text-foreground"
            }`}
          >
            {tab.label}
            {view === tab.id ? (
              <span className="absolute bottom-0 left-3 right-3 h-0.5 bg-accent" />
            ) : null}
          </button>
        ))}
      </div>

      {view === "kline" ? (
        <KlineToolbar
          options={options}
          resolution={resolution}
          onResolutionChange={onResolutionChange}
        />
      ) : null}

      <div className="relative min-h-0 flex-1 overflow-hidden">
        <div
          id="chart-panel-kline"
          role="tabpanel"
          className={`absolute inset-0 ${view === "kline" ? "block" : "invisible"}`}
        >
          <div className="flex h-full">
            <ChartTools />
            <div className="relative min-w-0 flex-1 bg-[#070a0b]">
              {displayData ? (
                <div className="pointer-events-none absolute left-3 top-2 z-10 select-none text-[11px]">
                  <div className="flex flex-wrap items-center gap-x-2 text-subtle">
                    <span>{symbol}USDT_SWAP · {options.find((item) => item.value === resolution)?.label ?? resolution} · MetaNode</span>
                    <span>开=<b className="font-medium text-foreground">{formatNumber(displayData.open, 2)}</b></span>
                    <span>高=<b className="font-medium text-foreground">{formatNumber(displayData.high, 2)}</b></span>
                    <span>低=<b className="font-medium text-foreground">{formatNumber(displayData.low, 2)}</b></span>
                    <span>收=<b className="font-medium text-foreground">{formatNumber(displayData.close, 2)}</b></span>
                    <span className={displayData.change >= 0 ? "text-buy" : "text-sell"}>
                      {formatNumber(displayData.close - displayData.open, 2)} ({formatPercent(displayData.change)})
                    </span>
                  </div>
                  <div className="mt-1 flex flex-wrap gap-x-4">
                    <span className="text-subtle">MA 5 close 0&nbsp; <b className="font-medium text-[#f3a11b]">{maValues.latest5 ? formatNumber(maValues.latest5, 2) : "—"}</b></span>
                    <span className="text-subtle">MA 10 close 0&nbsp; <b className="font-medium text-[#15b9df]">{maValues.latest10 ? formatNumber(maValues.latest10, 2) : "—"}</b></span>
                    <span className="text-subtle">MA 30 close 0&nbsp; <b className="font-medium text-[#cd63bd]">{maValues.latest30 ? formatNumber(maValues.latest30, 2) : "—"}</b></span>
                  </div>
                </div>
              ) : null}

              <div className="absolute right-2 top-2 z-10 flex items-center gap-1 text-[10px] text-subtle">
                <button type="button" className="flex items-center gap-1 border-0 bg-transparent px-2 py-1 hover:text-white">最新价格 <ChevronDownIcon className="h-3 w-3" /></button>
                <IconButton label="图表设置"><Cog6ToothIcon /></IconButton>
                <IconButton label="图表截图"><CameraIcon /></IconButton>
                <IconButton label="全屏图表"><ArrowsPointingOutIcon /></IconButton>
              </div>

              <div ref={containerRef} className="h-full w-full" />

              {loading ? (
                <div className="absolute inset-0 grid place-items-center bg-[#070a0b]/72 text-xs text-muted backdrop-blur-[1px]">K线加载中…</div>
              ) : null}
              {error ? (
                <div className="absolute bottom-10 left-1/2 -translate-x-1/2 rounded bg-sell/80 px-3 py-2 text-xs text-white">{error}</div>
              ) : null}
              <div className="pointer-events-none absolute bottom-1 left-12 text-[10px] text-subtle">日期范围⌄</div>
              <div className="pointer-events-none absolute bottom-1 right-3 text-[10px] text-subtle">UTC+8&nbsp;&nbsp;·&nbsp;&nbsp;%&nbsp;&nbsp;log&nbsp;&nbsp;<span className="text-white">自动</span></div>
            </div>
          </div>
        </div>

        {view === "info" ? (
          <InfoPanel id="chart-panel-info" data={data} symbol={symbol} latest={latestData} />
        ) : null}
        {view === "depth" ? (
          <DepthPanel id="chart-panel-depth" bids={depthBids} asks={depthAsks} />
        ) : null}
        {view === "trades" ? (
          <TradingDataPanel id="chart-panel-trades" trades={trades} symbol={symbol} />
        ) : null}
      </div>
    </div>
  );
}

function KlineToolbar({
  options,
  resolution,
  onResolutionChange,
}: {
  options: Option[];
  resolution: string;
  onResolutionChange: (value: string) => void;
}) {
  return (
    <div className="terminal-scrollbar flex h-9 shrink-0 items-center justify-between gap-3 overflow-x-auto border-b border-panelBorder px-2">
      <div className="flex items-center gap-0.5 whitespace-nowrap">
        <span className="px-2 text-[11px] text-subtle">分时</span>
        {options.map((option) => (
          <button
            key={option.value}
            type="button"
            onClick={() => onResolutionChange(option.value)}
            className={`rounded-sm border-0 px-2 py-1 text-[11px] font-medium ${
              option.value === resolution
                ? "bg-[#343940] text-white"
                : "bg-transparent text-subtle hover:text-white"
            }`}
          >
            {option.label}
          </button>
        ))}
        <span className="px-2 text-[11px] text-subtle">周线</span>
        <span className="flex items-center gap-0.5 px-2 text-[11px] text-subtle">1月 <ChevronDownIcon className="h-3 w-3" /></span>
      </div>
      <div className="flex shrink-0 items-center gap-1">
        <IconButton label="指标"><PresentationChartLineIcon /></IconButton>
        <IconButton label="蜡烛图"><ChartBarIcon /></IconButton>
        <IconButton label="绘图"><PencilSquareIcon /></IconButton>
      </div>
    </div>
  );
}

function ChartTools() {
  return (
    <div className="flex w-10 shrink-0 flex-col items-center gap-2 border-r border-panelBorder py-3 text-subtle">
      <IconButton label="十字光标"><CursorArrowRaysIcon /></IconButton>
      <IconButton label="趋势线"><PresentationChartLineIcon /></IconButton>
      <IconButton label="水平线"><MinusIcon /></IconButton>
      <IconButton label="图形工具"><Squares2X2Icon /></IconButton>
      <IconButton label="画笔"><PencilSquareIcon /></IconButton>
      <div className="my-1 h-px w-5 bg-panelBorder" />
      <IconButton label="放大"><MagnifyingGlassPlusIcon /></IconButton>
    </div>
  );
}

function IconButton({
  label,
  children,
}: {
  label: string;
  children: ReactElement<{ className?: string }>;
}) {
  return (
    <button
      type="button"
      aria-label={label}
      className="grid h-7 w-7 place-items-center rounded border-0 bg-transparent text-subtle hover:bg-elevated hover:text-white [&>svg]:h-4 [&>svg]:w-4"
    >
      {children}
    </button>
  );
}

function InfoPanel({
  id,
  data,
  symbol,
  latest,
}: {
  id: string;
  data: KlinePoint[];
  symbol: string;
  latest: (KlinePoint & { change: number }) | null;
}) {
  const stats = useMemo(() => {
    if (!data.length) return null;
    return {
      high: Math.max(...data.map((row) => row.high)),
      low: Math.min(...data.map((row) => row.low)),
      volume: data.reduce((sum, row) => sum + row.volume, 0),
    };
  }, [data]);

  return (
    <div id={id} role="tabpanel" className="market-tab-panel h-full overflow-y-auto bg-[#070a0b] p-5">
      <div className="text-sm font-semibold text-white">{symbol}USDT 永续合约信息</div>
      <p className="mt-1 text-[11px] text-subtle">MetaNode · USDC 保证金 · Sepolia 链上结算</p>
      <div className="mt-5 grid grid-cols-2 gap-px overflow-hidden border border-panelBorder bg-panelBorder lg:grid-cols-4">
        <Metric label="最新价格" value={latest ? formatNumber(latest.close, 2) : "—"} tone="buy" />
        <Metric label="区间最高" value={stats ? formatNumber(stats.high, 2) : "—"} />
        <Metric label="区间最低" value={stats ? formatNumber(stats.low, 2) : "—"} />
        <Metric label="区间成交量" value={stats ? formatNumber(stats.volume, 2) : "—"} />
      </div>
      <div className="mt-5 border border-panelBorder">
        {[
          ["合约类型", "永续"],
          ["保证金币种", "USDC"],
          ["价格来源", "Coinbase / OKX / Binance 加权指数"],
          ["结算网络", "Ethereum Sepolia"],
        ].map(([label, value]) => (
          <div key={label} className="flex items-center justify-between border-b border-panelBorder px-4 py-3 text-xs last:border-0">
            <span className="text-subtle">{label}</span>
            <span className="text-foreground">{value}</span>
          </div>
        ))}
      </div>
    </div>
  );
}

function Metric({ label, value, tone }: { label: string; value: string; tone?: "buy" }) {
  return (
    <div className="bg-panel px-4 py-4">
      <div className="text-[10px] text-subtle">{label}</div>
      <div className={`mt-2 font-mono text-lg font-semibold ${tone === "buy" ? "text-buy" : "text-white"}`}>{value}</div>
    </div>
  );
}

function DepthPanel({ id, bids, asks }: { id: string; bids: DepthLevel[]; asks: DepthLevel[] }) {
  return (
    <div id={id} role="tabpanel" className="market-tab-panel relative h-full bg-[#070a0b]">
      <div className="absolute left-4 top-3 z-10 flex items-center gap-5 text-[11px]">
        <span className="text-buy">买盘累计深度</span>
        <span className="text-sell">卖盘累计深度</span>
      </div>
      <DepthCanvas bids={bids} asks={asks} />
      {bids.length === 0 && asks.length === 0 ? (
        <div className="pointer-events-none absolute inset-0 grid place-items-center text-xs text-subtle">暂无订单簿深度数据</div>
      ) : null}
    </div>
  );
}

function DepthCanvas({ bids, asks }: { bids: DepthLevel[]; asks: DepthLevel[] }) {
  const canvasRef = useRef<HTMLCanvasElement>(null);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const draw = () => {
      const rect = canvas.getBoundingClientRect();
      const ratio = window.devicePixelRatio || 1;
      canvas.width = Math.max(1, Math.floor(rect.width * ratio));
      canvas.height = Math.max(1, Math.floor(rect.height * ratio));
      const ctx = canvas.getContext("2d");
      if (!ctx) return;
      ctx.setTransform(ratio, 0, 0, ratio, 0, 0);
      const width = rect.width;
      const height = rect.height;
      ctx.clearRect(0, 0, width, height);

      ctx.strokeStyle = "#182023";
      ctx.lineWidth = 1;
      for (let i = 1; i < 6; i += 1) {
        const x = (width / 6) * i;
        ctx.beginPath(); ctx.moveTo(x, 0); ctx.lineTo(x, height); ctx.stroke();
      }
      for (let i = 1; i < 5; i += 1) {
        const y = (height / 5) * i;
        ctx.beginPath(); ctx.moveTo(0, y); ctx.lineTo(width, y); ctx.stroke();
      }

      const maxAmount = Math.max(
        bids.reduce((sum, row) => sum + row.amount, 0),
        asks.reduce((sum, row) => sum + row.amount, 0),
        1
      );

      const drawSide = (levels: DepthLevel[], start: number, end: number, stroke: string, fill: string) => {
        if (!levels.length) return;
        let total = 0;
        const points = levels.slice(0, 40).map((level, index, rows) => {
          total += level.amount;
          return {
            x: start + ((end - start) * index) / Math.max(1, rows.length - 1),
            y: height - 32 - (total / maxAmount) * (height - 90),
          };
        });
        ctx.beginPath();
        ctx.moveTo(points[0].x, height - 32);
        points.forEach((point) => ctx.lineTo(point.x, point.y));
        ctx.lineTo(points[points.length - 1].x, height - 32);
        ctx.closePath();
        ctx.fillStyle = fill;
        ctx.fill();
        ctx.beginPath();
        points.forEach((point, index) => index === 0 ? ctx.moveTo(point.x, point.y) : ctx.lineTo(point.x, point.y));
        ctx.strokeStyle = stroke;
        ctx.lineWidth = 2;
        ctx.stroke();
      };

      drawSide([...bids].reverse(), 24, width / 2 - 5, "#13c991", "rgba(19,201,145,0.16)");
      drawSide(asks, width / 2 + 5, width - 24, "#f0446e", "rgba(240,68,110,0.16)");
    };

    draw();
    const observer = new ResizeObserver(draw);
    observer.observe(canvas);
    return () => observer.disconnect();
  }, [asks, bids]);

  return <canvas ref={canvasRef} className="h-full w-full" />;
}

function TradingDataPanel({ id, trades, symbol }: { id: string; trades: ChartTrade[]; symbol: string }) {
  const stats = useMemo(() => {
    const totalSize = trades.reduce((sum, trade) => sum + trade.size, 0);
    const buySize = trades.filter((trade) => trade.side === "BUY").reduce((sum, trade) => sum + trade.size, 0);
    const weighted = trades.reduce((sum, trade) => sum + trade.price * trade.size, 0);
    return {
      totalSize,
      buyPct: totalSize > 0 ? (buySize / totalSize) * 100 : 0,
      average: totalSize > 0 ? weighted / totalSize : 0,
    };
  }, [trades]);

  return (
    <div id={id} role="tabpanel" className="market-tab-panel h-full overflow-y-auto bg-[#070a0b] p-4">
      <div className="grid grid-cols-3 gap-px overflow-hidden border border-panelBorder bg-panelBorder">
        <Metric label="成交笔数" value={String(trades.length)} />
        <Metric label={`成交量(${symbol})`} value={formatNumber(stats.totalSize, 3)} />
        <Metric label="成交均价" value={stats.average > 0 ? formatNumber(stats.average, 2) : "—"} />
      </div>
      <div className="mt-4 flex h-2 overflow-hidden rounded-full bg-sell/20">
        <div className="bg-buy transition-[width] duration-500" style={{ width: `${stats.buyPct}%` }} />
      </div>
      <div className="mt-2 flex justify-between text-[10px]"><span className="text-buy">主动买入 {formatNumber(stats.buyPct, 2)}%</span><span className="text-sell">主动卖出 {formatNumber(100 - stats.buyPct, 2)}%</span></div>
      <div className="mt-5 grid grid-cols-3 border-b border-panelBorder pb-2 text-[10px] text-subtle"><span>价格(USDT)</span><span className="text-right">数量({symbol})</span><span className="text-right">时间</span></div>
      {trades.slice(0, 16).map((trade, index) => (
        <div key={`${trade.ts}-${index}`} className="trade-row-new grid h-7 grid-cols-3 items-center font-mono text-[11px]">
          <span className={trade.side === "BUY" ? "text-buy" : "text-sell"}>{formatNumber(trade.price, 2)}</span>
          <span className="text-right text-white">{formatNumber(trade.size, 3)}</span>
          <span className="text-right text-subtle">{formatTime(trade.ts * 1000)}</span>
        </div>
      ))}
      {trades.length === 0 ? <div className="grid h-40 place-items-center text-xs text-subtle">暂无交易数据</div> : null}
    </div>
  );
}
