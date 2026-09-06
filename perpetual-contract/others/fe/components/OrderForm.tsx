"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { ConnectButton } from "@rainbow-me/rainbowkit";
import {
  ArrowDownTrayIcon,
  ArrowsRightLeftIcon,
  BanknotesIcon,
  CalculatorIcon,
  ChevronDownIcon,
  ChevronRightIcon,
  InformationCircleIcon,
  TicketIcon,
} from "@heroicons/react/24/outline";
import { formatUnits } from "viem";
import { useAccount, useChainId, useReadContract, useSignTypedData } from "wagmi";
import { sepolia } from "wagmi/chains";
import {
  fetchMetanodeOpenQuote,
  fetchMetanodeOrderPreview,
  hasMetanodeAuthSession,
  postMetanodeOrder,
  type OrderPreviewDTO,
} from "@/lib/metanode-api";
import { useMetanodeAuth } from "@/lib/useMetanodeAuth";
import {
  applySlippageLimitPriceUsd,
  DEFAULT_SLIPPAGE_BPS,
  loadStoredSlippageBps,
  manualPriceExceedsSlippage,
  normalizeSlippageBps,
  saveStoredSlippageBps,
  SLIPPAGE_PRESETS_BPS,
  slippagePercentLabel,
} from "@/lib/slippage";
import {
  markPriceToUsd,
  type MetaNodeMarket,
} from "@/lib/metanode-markets";
import {
  buildMetaNodeOrder,
  orderTypedData,
  toCreateOrderBody,
  type OrderSide,
} from "@/lib/metanode-order";
import FundingRatePanel from "@/components/FundingRatePanel";
import { formatNumber } from "@/lib/format";
import {
  buildLeverageOptions,
  chainMinMarginUsd,
  clampLeverage,
  formatLeverageLabel,
  loadStoredLeverage,
  marginRequiredUsd,
  parseMaxLeverage,
  resolveMarketRisk,
  saveStoredLeverage,
} from "@/lib/leverage";
import type { PerpMarketDTO } from "@/lib/metanode-api";
import { metaNodeDealerAbi, resolveDealerAddress } from "@/lib/metanodeDealer";

export type CloseDraft = {
  perp: `0x${string}`;
  size: string;
  priceUsd: string;
  side: OrderSide;
};

type OpenPriceMode = "system" | "manual";

function formatOpenPriceSource(source: string): string {
  switch (source) {
    case "best_ask":
      return "订单簿卖一";
    case "best_bid":
      return "订单簿买一";
    case "orderbook_mid":
      return "订单簿中间价";
    case "chain_mark":
      return "链上标记价";
    case "spot_index":
      return "现货指数价";
    case "mark_fallback":
      return "标记价（本地）";
    default:
      return source || "系统价";
  }
}
type Props = {
  markets: MetaNodeMarket[];
  selectedPerp: `0x${string}`;
  onSelectPerp: (address: `0x${string}`) => void;
  markPriceRaw?: string;
  indexPriceUsd?: number;
  /** 交易页已选合约时可隐藏市场下拉 */
  hideMarketSelect?: boolean;
  /** 含链上杠杆配置的市场信息（来自 /markets） */
  marketRisk?: PerpMarketDTO;
  /** 当前订单簿买方数量占比（0–100） */
  bidSharePct?: number;
  dayLowUsd?: number;
  dayHighUsd?: number;
  closeDraft?: CloseDraft | null;
  onCloseDraftApplied?: () => void;
  onOrderSubmitted?: () => void;
};

export default function OrderForm({
  markets,
  selectedPerp,
  onSelectPerp,
  markPriceRaw,
  indexPriceUsd = 0,
  hideMarketSelect = false,
  marketRisk,
  bidSharePct,
  dayLowUsd,
  dayHighUsd,
  closeDraft,
  onCloseDraftApplied,
  onOrderSubmitted,
}: Props) {
  const { address, isConnected } = useAccount();
  const chainId = useChainId();
  const { signTypedDataAsync } = useSignTypedData();

  const dealerAddress = resolveDealerAddress();
  const [side, setSide] = useState<OrderSide>("long");
  const [price, setPrice] = useState("");
  const [openPriceSource, setOpenPriceSource] = useState<string>("");
  const [openPriceMode, setOpenPriceMode] = useState<OpenPriceMode>("system");
  const [amount, setAmount] = useState("");
  const [sizePercent, setSizePercent] = useState(0);
  const [submitting, setSubmitting] = useState(false);
  const [quoteLoading, setQuoteLoading] = useState(false);
  const [previewLoading, setPreviewLoading] = useState(false);
  const [referencePriceUsd, setReferencePriceUsd] = useState("");
  const [signLimitPriceUsd, setSignLimitPriceUsd] = useState("");
  const [slippageBps, setSlippageBps] = useState(DEFAULT_SLIPPAGE_BPS);
  const [preview, setPreview] = useState<OrderPreviewDTO | null>(null);
  const [msg, setMsg] = useState<string | null>(null);
  const [err, setErr] = useState<string | null>(null);

  const markUsd = useMemo(() => markPriceToUsd(markPriceRaw), [markPriceRaw]);
  const [panelMode, setPanelMode] = useState<"open" | "close">("open");
  const isCloseMode = panelMode === "close" || Boolean(closeDraft);
  const onSepolia = chainId === sepolia.id;
  const { sessionOk, authPending } = useMetanodeAuth(address);

  const selectedMarket = markets.find(
    (m) => m.address.toLowerCase() === selectedPerp.toLowerCase()
  );
  const risk = resolveMarketRisk(marketRisk);
  const maxLeverageNum = parseMaxLeverage(risk.maxLeverage);
  const leverageOptions = useMemo(
    () => buildLeverageOptions(maxLeverageNum),
    [maxLeverageNum]
  );
  const [leverage, setLeverage] = useState(10);

  useEffect(() => {
    setSlippageBps(loadStoredSlippageBps());
  }, []);

  useEffect(() => {
    if (isCloseMode) return;
    setLeverage(loadStoredLeverage(selectedPerp, maxLeverageNum));
  }, [selectedPerp, maxLeverageNum, isCloseMode]);

  const handleLeverageChange = (lev: number) => {
    const next = clampLeverage(lev, maxLeverageNum);
    setLeverage(next);
    saveStoredLeverage(selectedPerp, next);
  };

  const applyReferencePrice = useCallback(
    (refUsd: string, source: string) => {
      setReferencePriceUsd(refUsd);
      setOpenPriceSource(source);
      const limit = applySlippageLimitPriceUsd(refUsd, side, slippageBps);
      setSignLimitPriceUsd(limit);
      setPrice(refUsd);
    },
    [side, slippageBps]
  );

  const refreshSystemQuote = useCallback(async () => {
    setQuoteLoading(true);
    try {
      const resp = await fetchMetanodeOpenQuote(selectedPerp, side);
      if (resp.code === 0 && resp.quote?.priceUsd) {
        applyReferencePrice(resp.quote.priceUsd, resp.quote.source ?? "");
        return;
      }
      if (markUsd > 0) {
        applyReferencePrice(formatNumber(markUsd, 2), "mark_fallback");
      }
    } catch {
      if (markUsd > 0) {
        applyReferencePrice(formatNumber(markUsd, 2), "mark_fallback");
      }
    } finally {
      setQuoteLoading(false);
    }
  }, [selectedPerp, side, markUsd, applyReferencePrice]);

  useEffect(() => {
    if (!closeDraft) return;
    setPanelMode("close");
    setSide(closeDraft.side);
    setAmount(closeDraft.size);
    setOpenPriceMode("system");
    void refreshSystemQuote();
    onCloseDraftApplied?.();
  }, [closeDraft, onCloseDraftApplied, refreshSystemQuote]);

  useEffect(() => {
    if (referencePriceUsd) {
      setSignLimitPriceUsd(
        applySlippageLimitPriceUsd(referencePriceUsd, side, slippageBps)
      );
    }
  }, [slippageBps, referencePriceUsd, side]);

  useEffect(() => {
    if (!amount.trim()) {
      setPreview(null);
      return;
    }
    const timer = window.setTimeout(async () => {
      setPreviewLoading(true);
      try {
        const resp = await fetchMetanodeOrderPreview(
          selectedPerp,
          side,
          amount.trim(),
          slippageBps,
          address
        );
        if (resp.code === 0 && resp.preview) {
          setPreview(resp.preview);
          if (resp.preview.referencePriceUsd) {
            setReferencePriceUsd(resp.preview.referencePriceUsd);
            setSignLimitPriceUsd(resp.preview.limitPriceUsd);
          }
        } else {
          setPreview(null);
        }
      } catch {
        setPreview(null);
      } finally {
        setPreviewLoading(false);
      }
    }, 350);
    return () => window.clearTimeout(timer);
  }, [selectedPerp, side, amount, slippageBps, address]);

  useEffect(() => {
    if (isCloseMode || openPriceMode !== "system") return;
    void refreshSystemQuote();
  }, [isCloseMode, openPriceMode, refreshSystemQuote]);

  useEffect(() => {
    if (openPriceMode === "system" && !isCloseMode) return;
    void (async () => {
      try {
        const resp = await fetchMetanodeOpenQuote(selectedPerp, side);
        if (resp.code === 0 && resp.quote?.priceUsd) {
          setReferencePriceUsd(resp.quote.priceUsd);
          setOpenPriceSource(resp.quote.source ?? "");
        }
      } catch {
        /* ignore */
      }
    })();
  }, [selectedPerp, side, openPriceMode, isCloseMode]);

  const handleSlippageChange = (bps: number) => {
    const next = normalizeSlippageBps(bps);
    setSlippageBps(next);
    saveStoredSlippageBps(next);
  };

  const resolveSignPriceUsd = useCallback(
    (orderSide: OrderSide = side): string | null => {
      if (!isCloseMode && openPriceMode === "manual") {
        const manual = price.trim();
        if (
          referencePriceUsd &&
          manualPriceExceedsSlippage(
            manual,
            referencePriceUsd,
            orderSide,
            slippageBps
          )
        ) {
          return null;
        }
        return manual;
      }
      if (signLimitPriceUsd.trim()) return signLimitPriceUsd.trim();
      if (referencePriceUsd.trim()) {
        return applySlippageLimitPriceUsd(
          referencePriceUsd,
          orderSide,
          slippageBps
        );
      }
      return price.trim() || null;
    },
    [
      isCloseMode,
      openPriceMode,
      price,
      referencePriceUsd,
      side,
      slippageBps,
      signLimitPriceUsd,
    ]
  );

  const useSystemOpenPrice =
    (!isCloseMode && openPriceMode === "system") ||
    (isCloseMode && Boolean(closeDraft));

  const notional = useMemo(() => {
    const p = Number(price) || markUsd || 0;
    const a = Number(amount) || 0;
    return p * a;
  }, [price, amount, markUsd]);

  const { data: creditData } = useReadContract({
    address: dealerAddress,
    abi: metaNodeDealerAbi,
    functionName: "getCreditOf",
    args: address ? [address] : undefined,
    chainId: sepolia.id,
    query: { enabled: Boolean(address && onSepolia) },
  });

  const dealerCreditHuman = useMemo(() => {
    if (!creditData) return null;
    const raw = (creditData as readonly [bigint, ...unknown[]])[0];
    return Number(formatUnits(raw, 6));
  }, [creditData]);

  const marginRequired = useMemo(
    () => (isCloseMode ? 0 : marginRequiredUsd(notional, leverage)),
    [isCloseMode, notional, leverage]
  );

  const chainMinMargin = useMemo(
    () =>
      isCloseMode ? 0 : chainMinMarginUsd(notional, risk.initialMarginPct),
    [isCloseMode, notional, risk.initialMarginPct]
  );

  const creditInsufficient = useMemo(() => {
    if (isCloseMode || dealerCreditHuman == null || marginRequired <= 0) {
      return false;
    }
    return dealerCreditHuman < marginRequired;
  }, [isCloseMode, dealerCreditHuman, marginRequired]);

  const leverageTooLowForChain = useMemo(() => {
    if (isCloseMode || marginRequired <= 0 || chainMinMargin <= 0) return false;
    return marginRequired < chainMinMargin - 1e-6;
  }, [isCloseMode, marginRequired, chainMinMargin]);

  const submit = useCallback(async (submitSide: OrderSide = side) => {
    setErr(null);
    setMsg(null);
    if (!address || !isConnected) {
      setErr("请先连接钱包");
      return;
    }
    if (!onSepolia) {
      setErr("请切换到 Sepolia");
      return;
    }
    if (!hasMetanodeAuthSession(address)) {
      setErr(
        authPending
          ? "MetaNode 登录进行中，请先在钱包完成登录签名（与连接钱包不同）"
          : "MetaNode 登录未完成：请刷新页面，并在钱包中完成 MetaNode 登录签名后再平仓"
      );
      return;
    }
    if (!amount.trim() || !price.trim()) {
      setErr("请填写数量与价格");
      return;
    }
    const signPriceUsd = resolveSignPriceUsd(submitSide);
    if (!signPriceUsd) {
      setErr(
        `手动限价超出滑点保护（${slippagePercentLabel(slippageBps)}），请提高滑点或调整价格`
      );
      return;
    }
    if (!isCloseMode && leverage > maxLeverageNum) {
      setErr(`杠杆不能超过该合约最大 ${maxLeverageNum}x`);
      return;
    }
    if (!isCloseMode && creditInsufficient) {
      setErr(
        `Dealer 保证金不足：需要约 ${formatNumber(marginRequired, 2)} USDC（${leverage}x）`
      );
      return;
    }
    if (!isCloseMode && leverageTooLowForChain) {
      setErr(
        `所选杠杆过高：${leverage}x 保证金低于链上最低要求（约 ${formatNumber(chainMinMargin, 2)} USDC）`
      );
      return;
    }

    setSubmitting(true);
    try {
      const order = buildMetaNodeOrder({
        perp: selectedPerp,
        signer: address,
        side: submitSide,
        size: amount.trim(),
        priceUsd: signPriceUsd,
      });
      const typed = orderTypedData(order);
      const signature = await signTypedDataAsync(typed);
      const resp = await postMetanodeOrder(toCreateOrderBody(order, signature));
      if (resp.code !== 0) {
        setErr(resp.message || `下单失败 code=${resp.code}`);
        return;
      }
      setMsg(
        resp.orderId
          ? `订单已提交：${resp.orderId.slice(0, 10)}…`
          : "订单已提交，等待撮合"
      );
      onOrderSubmitted?.();
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setSubmitting(false);
    }
  }, [
    address,
    isConnected,
    onSepolia,
    authPending,
    resolveSignPriceUsd,
    slippageBps,
    amount,
    price,
    selectedPerp,
    side,
    signTypedDataAsync,
    onOrderSubmitted,
    isCloseMode,
    leverage,
    maxLeverageNum,
    marginRequired,
    creditInsufficient,
    leverageTooLowForChain,
  ]);

  const priceNum = Number(price) || 0;
  const maxSizeByCredit = useMemo(() => {
    const reference = priceNum || markUsd || indexPriceUsd;
    if (isCloseMode || !reference || dealerCreditHuman == null) return 0;
    return (dealerCreditHuman * leverage) / reference;
  }, [dealerCreditHuman, indexPriceUsd, isCloseMode, leverage, markUsd, priceNum]);

  const handleSizePercentChange = (nextPercent: number) => {
    setSizePercent(nextPercent);
    if (maxSizeByCredit > 0) {
      setAmount(((maxSizeByCredit * nextPercent) / 100).toFixed(4));
    }
  };

  if (isConnected) {
    const baseSymbol = selectedMarket?.symbol ?? "BTC";
    const referenceUsd = priceNum || markUsd || indexPriceUsd;
    const rangeLow = dayLowUsd && dayLowUsd > 0 ? dayLowUsd : referenceUsd;
    const rangeHigh = dayHighUsd && dayHighUsd > 0 ? dayHighUsd : referenceUsd;
    const rangePosition =
      rangeHigh > rangeLow
        ? Math.min(
            100,
            Math.max(0, ((referenceUsd - rangeLow) / (rangeHigh - rangeLow)) * 100)
          )
        : 50;
    const riskRate =
      marginRequired > 0 && dealerCreditHuman != null
        ? Math.min(999, (dealerCreditHuman / marginRequired) * 100)
        : 999;

    return (
      <div className="min-h-full bg-[#070a0b] text-foreground">
        <div className="px-4 pb-7 pt-3">
          <div className="relative flex h-11 items-center gap-2 rounded-sm bg-[#2c3139] px-3 text-sm text-white">
            <span className="font-semibold">逐仓</span>
            <span className="rounded bg-accent/15 px-2 py-1 font-mono text-xs font-bold text-accent">
              {leverage}X
            </span>
            <select
              aria-label="选择杠杆倍数"
              value={leverage}
              onChange={(event) => handleLeverageChange(Number(event.target.value))}
              className="absolute inset-0 cursor-pointer opacity-0"
            >
              {leverageOptions.map((option) => (
                <option key={option} value={option}>
                  {option}X
                </option>
              ))}
            </select>
          </div>

          <div className="mt-4 grid grid-cols-2 overflow-hidden rounded bg-[#181d21]">
            <button
              type="button"
              onClick={() => setSide("long")}
              className={
                "h-11 rounded border bg-transparent text-sm font-bold transition " +
                (side === "long"
                  ? "border-buy text-buy"
                  : "border-transparent text-subtle hover:text-white")
              }
            >
              买进
            </button>
            <button
              type="button"
              onClick={() => setSide("short")}
              className={
                "h-11 rounded border bg-transparent text-sm font-bold transition " +
                (side === "short"
                  ? "border-sell text-sell"
                  : "border-transparent text-subtle hover:text-white")
              }
            >
              卖出
            </button>
          </div>

          <div className="mt-5 flex items-center gap-1">
            <button
              type="button"
              onClick={() => {
                setOpenPriceMode("manual");
                if (!price.trim() && referenceUsd > 0) {
                  setPrice(referenceUsd.toFixed(2));
                }
              }}
              className={
                "rounded border-0 px-3 py-2 text-sm font-semibold " +
                (openPriceMode === "manual"
                  ? "bg-[#2c3139] text-white"
                  : "bg-transparent text-subtle")
              }
            >
              限价
            </button>
            <button
              type="button"
              onClick={() => setOpenPriceMode("system")}
              className={
                "rounded border-0 px-3 py-2 text-sm font-semibold " +
                (openPriceMode === "system"
                  ? "bg-[#2c3139] text-white"
                  : "bg-transparent text-subtle")
              }
            >
              市价
            </button>
            <button
              type="button"
              disabled
              className="border-0 bg-transparent px-3 py-2 text-sm font-semibold text-subtle"
            >
              高级限价委托
            </button>
            <InformationCircleIcon className="ml-auto h-4 w-4 text-subtle" />
          </div>

          <div className="mt-3 flex items-center gap-1">
            <button
              type="button"
              className="rounded border-0 bg-[#2c3139] px-3 py-2 text-xs font-semibold text-white"
            >
              普通
            </button>
            <button
              type="button"
              disabled
              className="border-0 bg-transparent px-3 py-2 text-xs font-semibold text-subtle"
            >
              自动借款
            </button>
            <button
              type="button"
              disabled
              className="border-0 bg-transparent px-3 py-2 text-xs font-semibold text-subtle"
            >
              自动还款
            </button>
            <InformationCircleIcon className="ml-auto h-4 w-4 text-subtle" />
          </div>

          <div className="mt-3 flex items-center gap-2 text-xs text-subtle">
            <span>风险率</span>
            <span
              className="relative h-4 w-4 rounded-full"
              style={{
                background:
                  "conic-gradient(#f6c945 0 25%, #16d8d4 25% 72%, #394148 72% 100%)",
              }}
            >
              <span className="absolute inset-[3px] rounded-full bg-[#070a0b]" />
            </span>
            <span className="font-mono text-sm font-medium text-buy">
              {formatNumber(riskRate, 2)}
            </span>
          </div>

          <div className="mt-4 space-y-3">
            <div className="relative h-[58px] rounded bg-[#1d2227] px-3 pb-2 pt-2">
              <div className="text-[11px] text-subtle">单价</div>
              {openPriceMode === "system" ? (
                <div className="mt-1 truncate pr-14 text-sm font-semibold text-white">
                  以市场最优价格{side === "long" ? "买入" : "卖出"}
                </div>
              ) : (
                <input
                  value={price}
                  onChange={(event) => setPrice(event.target.value)}
                  inputMode="decimal"
                  aria-label={isCloseMode ? "平仓限价" : "开仓限价"}
                  placeholder={referenceUsd > 0 ? referenceUsd.toFixed(2) : "0.00"}
                  className="mt-0.5 w-[calc(100%-60px)] border-0 bg-transparent p-0 font-mono text-lg text-white outline-none placeholder:text-faint"
                />
              )}
              <span className="absolute bottom-3 right-3 text-sm font-semibold text-white">
                USDT
              </span>
            </div>

            <label className="relative block h-[58px] rounded bg-[#1d2227] px-3 pb-2 pt-2">
              <span className="block text-[11px] text-subtle">数量</span>
              <input
                value={amount}
                onChange={(event) => {
                  setAmount(event.target.value);
                  setSizePercent(0);
                }}
                inputMode="decimal"
                aria-label="订单数量"
                placeholder="最小交易数量 0.001"
                className="mt-0.5 w-[calc(100%-58px)] border-0 bg-transparent p-0 font-mono text-base text-white outline-none placeholder:text-faint"
              />
              <span className="absolute bottom-3 right-3 text-sm font-semibold text-white">
                {baseSymbol}
              </span>
            </label>

            <div className="relative px-1 pt-2">
              <input
                type="range"
                min={0}
                max={100}
                step={25}
                value={sizePercent}
                aria-label="仓位数量百分比"
                onChange={(event) =>
                  handleSizePercentChange(Number(event.target.value))
                }
                className="order-size-range w-full"
                style={{
                  background:
                    "linear-gradient(to right, #16d8d4 0%, #16d8d4 " +
                    sizePercent +
                    "%, #343a40 " +
                    sizePercent +
                    "%, #343a40 100%)",
                }}
              />
              <div className="pointer-events-none absolute left-1 right-1 top-[15px] flex justify-between px-[2px]">
                {[0, 25, 50, 75, 100].map((tick) => (
                  <span
                    key={tick}
                    className={
                      "h-1.5 w-1.5 rounded-full " +
                      (tick <= sizePercent ? "bg-accent" : "bg-[#747f82]")
                    }
                  />
                ))}
              </div>
              <div className="mt-1 flex justify-between text-[11px] text-subtle">
                <span>{sizePercent}%</span>
                <span />
              </div>
            </div>

            <label className="relative block h-[58px] rounded bg-[#1d2227] px-3 pb-2 pt-2">
              <span className="block text-[11px] text-subtle">金额</span>
              <input
                value={notional > 0 ? notional.toFixed(2) : ""}
                onChange={(event) => {
                  const total = Number(event.target.value);
                  if (referenceUsd > 0 && Number.isFinite(total)) {
                    setAmount((total / referenceUsd).toFixed(4));
                    setSizePercent(0);
                  }
                }}
                inputMode="decimal"
                aria-label="订单金额"
                placeholder="> 5 USDT"
                className="mt-0.5 w-[calc(100%-58px)] border-0 bg-transparent p-0 font-mono text-base text-white outline-none placeholder:text-faint"
              />
              <span className="absolute bottom-3 right-3 text-sm font-semibold text-white">
                USDT
              </span>
            </label>
          </div>

          <div className="mt-3 flex items-center text-xs text-subtle">
            <span>
              可用
              <b className="ml-1 font-mono font-medium text-white">
                {dealerCreditHuman != null
                  ? formatNumber(dealerCreditHuman, 2)
                  : "0.00"} {" "}
                USDC
              </b>
            </span>
            <span className="ml-2 rounded-full border border-panelBorder px-1 text-[9px] text-white">
              Z
            </span>
            <BanknotesIcon className="ml-3 h-4 w-4 text-white" />
            <ArrowDownTrayIcon className="ml-auto h-4 w-4 text-white" />
            <ArrowsRightLeftIcon className="ml-4 h-4 w-4 text-white" />
          </div>

          <button
            type="button"
            disabled={
              submitting ||
              authPending ||
              !amount.trim() ||
              (!isCloseMode &&
                (creditInsufficient || leverageTooLowForChain))
            }
            onClick={() => void submit(side)}
            className={
              "mt-6 h-12 w-full rounded border-0 text-sm font-bold text-white transition hover:brightness-110 disabled:cursor-not-allowed disabled:opacity-40 " +
              (side === "long" ? "bg-buy" : "bg-sell")
            }
          >
            {submitting
              ? "提交中…"
              : `${side === "long" ? "买进" : "卖出"} ${baseSymbol}`}
          </button>

          <button
            type="button"
            className="ml-auto mt-3 flex items-center gap-1 border-0 bg-transparent text-xs text-subtle hover:text-white"
          >
            费率
            <ChevronRightIcon className="h-3.5 w-3.5" />
          </button>

          {authPending ? (
            <p className="mt-3 text-[11px] text-amber-400">
              请在钱包中完成 MetaNode 登录签名…
            </p>
          ) : null}
          {err ? <p className="mt-3 text-[11px] text-sell">{err}</p> : null}
          {msg ? <p className="mt-3 text-[11px] text-buy">{msg}</p> : null}
        </div>

        <section className="border-t border-panelBorder px-4 pb-8 pt-5">
          <div className="flex items-center gap-6">
            <h3 className="text-base font-bold text-white">交易数据</h3>
            <span className="relative text-sm font-semibold text-subtle">
              市场异动
              <span className="absolute -right-2 -top-1 h-1.5 w-1.5 rounded-full bg-sell" />
            </span>
          </div>

          <div className="mt-5 flex items-center gap-2 text-xs">
            <span className="font-semibold text-white">24小时 价格区间</span>
            <span className="text-subtle">24小时</span>
            <ChevronDownIcon className="h-3 w-3 text-subtle" />
          </div>

          <div className="relative mt-9 h-1.5 rounded-full bg-[#343a40]">
            <span
              className="absolute -top-8 -translate-x-1/2 rounded bg-[#2c3139] px-2 py-1 text-[11px] font-semibold text-white after:absolute after:left-1/2 after:top-full after:-translate-x-1/2 after:border-4 after:border-transparent after:border-t-[#2c3139]"
              style={{ left: `${rangePosition}%` }}
            >
              当前价格
            </span>
            <span
              className="absolute -top-1 h-3.5 w-1 rounded-full bg-accent"
              style={{ left: `${rangePosition}%` }}
            />
          </div>

          <div className="mt-3 flex justify-between text-xs text-subtle">
            <div>
              <div>24小时 最低价</div>
              <div className="mt-1 font-mono font-medium text-white">
                {rangeLow > 0 ? formatNumber(rangeLow, 2) : "—"}
              </div>
            </div>
            <div className="text-right">
              <div>24小时 最高价</div>
              <div className="mt-1 font-mono font-medium text-white">
                {rangeHigh > 0 ? formatNumber(rangeHigh, 2) : "—"}
              </div>
            </div>
          </div>

          <div className="mt-7 grid grid-cols-2 gap-5 text-xs">
            <div>
              <div className="text-subtle">成交用户占比 (24小时)</div>
              <div className="mt-2 font-mono font-semibold text-white">
                {bidSharePct != null ? `${formatNumber(bidSharePct, 2)}%` : "—"}
              </div>
            </div>
            <div>
              <div className="text-subtle">交易选择倾向 (24小时)</div>
              <div className={"mt-2 font-semibold " + (side === "long" ? "text-buy" : "text-sell")}>
                {side === "long" ? "买" : "卖"}
              </div>
            </div>
            <MarketGauge
              label="买卖人数占比 (24小时)"
              buyShare={bidSharePct}
            />
            <MarketGauge
              label="买卖挂单金额占比 (24小时)"
              buyShare={bidSharePct}
            />
          </div>
        </section>
      </div>
    );
  }

  return (
    <div className="min-h-full bg-[#070a0b] text-foreground">
      <div className="flex h-14 items-end gap-7 border-b border-panelBorder px-5">
        <button
          type="button"
          className="relative h-full border-0 bg-transparent pt-2 text-base font-bold text-white after:absolute after:bottom-0 after:left-0 after:right-0 after:h-0.5 after:bg-white"
        >
          交易
        </button>
        <button
          type="button"
          disabled
          className="h-full border-0 bg-transparent pt-2 text-base font-semibold text-subtle"
        >
          机器人
        </button>
      </div>

      <div className="px-4 pb-5 pt-4">
        <div className="flex items-center gap-2">
          <button
            type="button"
            className="flex h-9 items-center gap-1.5 rounded bg-[#2c3139] px-3 text-sm font-semibold text-white hover:bg-[#383e47]"
          >
            全仓
            <ChevronDownIcon className="h-3.5 w-3.5 text-subtle" />
          </button>

          <label className="relative flex h-9 items-center gap-1.5 rounded bg-[#2c3139] px-3 text-sm font-semibold text-white hover:bg-[#383e47]">
            {leverage}X
            <ChevronDownIcon className="h-3.5 w-3.5 text-subtle" />
            <select
              aria-label="选择杠杆倍数"
              value={leverage}
              onChange={(event) => handleLeverageChange(Number(event.target.value))}
              className="absolute inset-0 cursor-pointer opacity-0"
            >
              {leverageOptions.map((option) => (
                <option key={option} value={option}>
                  {option}X
                </option>
              ))}
            </select>
          </label>

          <button
            type="button"
            aria-label="合约体验金"
            className="relative ml-auto grid h-9 w-9 place-items-center border-0 bg-transparent text-white hover:text-accent"
          >
            <TicketIcon className="h-6 w-6" />
            <span className="absolute right-0.5 top-0.5 h-2 w-2 rounded-full border border-[#070a0b] bg-sell" />
          </button>
        </div>

        <div className="mt-5 grid grid-cols-2 overflow-hidden rounded border border-panelBorder bg-[#252a31]">
          <button
            type="button"
            onClick={() => setPanelMode("open")}
            className={
              "h-11 border-0 text-sm font-bold " +
              (panelMode === "open"
                ? "rounded border border-buy bg-[#252a31] text-buy"
                : "bg-transparent text-subtle hover:text-white")
            }
          >
            开仓
          </button>
          <button
            type="button"
            onClick={() => setPanelMode("close")}
            className={
              "h-11 border-0 text-sm font-bold " +
              (panelMode === "close"
                ? "rounded border border-sell bg-[#252a31] text-sell"
                : "bg-transparent text-subtle hover:text-white")
            }
          >
            平仓
          </button>
        </div>

        {!hideMarketSelect ? (
          <label className="mt-4 block text-xs text-subtle">
            市场
            <select
              value={selectedPerp}
              onChange={(event) =>
                onSelectPerp(event.target.value as `0x${string}`)
              }
              className="mt-1.5 w-full rounded border border-panelBorder bg-[#2c3139] px-3 py-2.5 text-sm text-white outline-none"
            >
              {markets.map((market) => (
                <option key={market.address} value={market.address}>
                  {market.name}
                </option>
              ))}
            </select>
          </label>
        ) : null}

        <div className="mt-5 flex items-center gap-1">
          <button
            type="button"
            onClick={() => {
              setOpenPriceMode("manual");
              if (!price.trim() && (markUsd > 0 || indexPriceUsd > 0)) {
                setPrice(formatNumber(markUsd || indexPriceUsd, 2));
              }
            }}
            className={
              "rounded border-0 px-3 py-2 text-sm font-semibold " +
              (openPriceMode === "manual"
                ? "bg-[#2c3139] text-white"
                : "bg-transparent text-subtle hover:text-white")
            }
          >
            限价
          </button>
          <button
            type="button"
            onClick={() => setOpenPriceMode("system")}
            className={
              "rounded border-0 px-3 py-2 text-sm font-semibold " +
              (openPriceMode === "system"
                ? "bg-[#2c3139] text-white"
                : "bg-transparent text-subtle hover:text-white")
            }
          >
            市价
          </button>
          <button
            type="button"
            disabled
            className="flex items-center gap-1 border-0 bg-transparent px-3 py-2 text-sm font-semibold text-subtle"
          >
            计划
            <ChevronDownIcon className="h-3.5 w-3.5" />
          </button>
          <InformationCircleIcon className="ml-1 h-4 w-4 text-subtle" />
        </div>

        <button
          type="button"
          onClick={() =>
            handleSlippageChange(
              SLIPPAGE_PRESETS_BPS[
                (SLIPPAGE_PRESETS_BPS.findIndex(
                  (preset) => preset === slippageBps
                ) +
                  1) %
                  SLIPPAGE_PRESETS_BPS.length
              ]
            )
          }
          className="relative mt-3 flex h-7 w-full items-center justify-center rounded-full border-0 bg-amber-700/55 px-4 text-[11px] font-semibold text-amber-400 hover:bg-amber-700/70"
        >
          限制最大滑点 {slippagePercentLabel(slippageBps)}
          <InformationCircleIcon className="ml-1.5 h-3.5 w-3.5" />
          <span className="absolute -top-1 left-24 h-2 w-2 rotate-45 bg-amber-700/55" />
        </button>

        <div className="mt-5 flex items-center text-xs text-subtle">
          <span>
            可用余额:
            <b className="ml-0.5 font-mono font-medium text-white">
              {dealerCreditHuman != null
                ? formatNumber(dealerCreditHuman, 2)
                : "0.00"}{" "}
              USDC
            </b>
          </span>
          <span
            className={
              "ml-2 rounded-full border px-1 text-[9px] " +
              (sessionOk
                ? "border-buy/50 text-buy"
                : "border-panelBorder text-subtle")
            }
          >
            Z
          </span>
          <CalculatorIcon className="ml-auto h-4 w-4 text-subtle" />
        </div>

        <div className="mt-3 grid grid-cols-[minmax(0,1fr)_58px] gap-2">
          <div className="relative h-[58px] rounded bg-[#2c3139] px-3 pb-2 pt-2">
            <div className="text-[11px] text-subtle">
              单价
              {quoteLoading ? " · 刷新中" : ""}
            </div>
            {useSystemOpenPrice ? (
              <div className="mt-0.5 font-mono text-xl font-medium text-white">
                {priceNum > 0 ? formatNumber(priceNum, 2) : "—"}
              </div>
            ) : (
              <input
                value={price}
                onChange={(event) => setPrice(event.target.value)}
                inputMode="decimal"
                aria-label={isCloseMode ? "平仓限价" : "开仓限价"}
                placeholder={markUsd > 0 ? formatNumber(markUsd, 2) : "65455.9"}
                className="mt-0.5 w-[calc(100%-58px)] border-0 bg-transparent p-0 font-mono text-xl text-white outline-none placeholder:text-faint"
              />
            )}
            <span className="absolute bottom-3 right-3 text-sm font-semibold text-white">
              USDT
            </span>
          </div>
          <button
            type="button"
            disabled={markUsd <= 0 && indexPriceUsd <= 0}
            onClick={() => {
              setOpenPriceMode("manual");
              setPrice(formatNumber(markUsd || indexPriceUsd, 2));
            }}
            className="rounded border-0 bg-[#2c3139] text-sm font-semibold text-subtle hover:text-white disabled:opacity-40"
          >
            BBO
          </button>
        </div>

        <label className="relative mt-3 block h-[58px] rounded bg-[#2c3139] px-3 pb-2 pt-2">
          <div className="text-[11px] text-subtle">数量</div>
          <input
            value={amount}
            onChange={(event) => {
              setAmount(event.target.value);
              setSizePercent(0);
            }}
            inputMode="decimal"
            aria-label="订单数量"
            placeholder="最小交易数量 0.001"
            className="mt-0.5 w-[calc(100%-58px)] border-0 bg-transparent p-0 font-mono text-base text-white outline-none placeholder:text-faint"
          />
          <span className="absolute bottom-3 right-3 flex items-center gap-1 text-sm font-semibold text-white">
            {selectedMarket?.symbol ?? "BTC"}
            <ChevronDownIcon className="h-3.5 w-3.5 text-subtle" />
          </span>
        </label>

        <div className="relative mt-5 px-1">
          <input
            type="range"
            min={0}
            max={100}
            step={25}
            value={sizePercent}
            aria-label="仓位数量百分比"
            onChange={(event) =>
              handleSizePercentChange(Number(event.target.value))
            }
            className="order-size-range w-full"
            style={{
              background:
                "linear-gradient(to right, #16d8d4 0%, #16d8d4 " +
                sizePercent +
                "%, #343a40 " +
                sizePercent +
                "%, #343a40 100%)",
            }}
          />
          <div className="pointer-events-none absolute left-1 right-1 top-[7px] flex justify-between px-[2px]">
            {[0, 25, 50, 75, 100].map((tick) => (
              <span
                key={tick}
                className={
                  "h-1.5 w-1.5 rounded-full " +
                  (tick <= sizePercent ? "bg-accent" : "bg-[#747f82]")
                }
              />
            ))}
          </div>
          <div className="mt-1 flex justify-between text-[11px] text-subtle">
            <span>0</span>
            <span>{sizePercent}%</span>
          </div>
        </div>

        {isConnected ? (
          <div className="mt-5 grid grid-cols-2 gap-2">
            <button
              type="button"
              disabled={
                submitting ||
                authPending ||
                (panelMode === "close" && !closeDraft && !amount) ||
                (!isCloseMode &&
                  (creditInsufficient || leverageTooLowForChain))
              }
              onClick={() => {
                setSide("long");
                void submit("long");
              }}
              className="h-12 rounded border-0 bg-buy text-sm font-bold text-[#03120d] transition hover:brightness-110 disabled:cursor-not-allowed disabled:opacity-40"
            >
              {submitting
                ? "提交中…"
                : isCloseMode
                  ? "买入 / 平空"
                  : "买入 / 做多"}
            </button>
            <button
              type="button"
              disabled={
                submitting ||
                authPending ||
                (panelMode === "close" && !closeDraft && !amount) ||
                (!isCloseMode &&
                  (creditInsufficient || leverageTooLowForChain))
              }
              onClick={() => {
                setSide("short");
                void submit("short");
              }}
              className="h-12 rounded border-0 bg-sell text-sm font-bold text-white transition hover:brightness-110 disabled:cursor-not-allowed disabled:opacity-40"
            >
              {submitting
                ? "提交中…"
                : isCloseMode
                  ? "卖出 / 平多"
                  : "卖出 / 做空"}
            </button>
          </div>
        ) : (
          <div className="mt-6 space-y-3">
            <Link
              href="/faucet"
              className="flex h-12 items-center justify-center gap-2 rounded bg-accent text-sm font-bold text-[#041112] hover:bg-[#28e4df]"
            >
              <TicketIcon className="h-5 w-5" />
              领取 Sepolia 测试 USDC
            </Link>
            <ConnectButton.Custom>
              {({ openConnectModal }) => (
                <button
                  type="button"
                  onClick={openConnectModal}
                  className="h-12 w-full rounded border border-[#343a43] bg-transparent text-sm font-bold text-white hover:border-accent hover:text-accent"
                >
                  连接钱包
                </button>
              )}
            </ConnectButton.Custom>
          </div>
        )}

        {panelMode === "close" && !closeDraft ? (
          <p className="mt-3 text-[11px] text-amber-400">
            请从下方仓位列表选择需要平仓的仓位。
          </p>
        ) : null}
        {authPending && isConnected ? (
          <p className="mt-3 text-[11px] text-amber-400">
            请在钱包中完成 MetaNode 登录签名…
          </p>
        ) : null}
        {err ? <p className="mt-3 text-[11px] text-sell">{err}</p> : null}
        {msg ? <p className="mt-3 text-[11px] text-buy">{msg}</p> : null}
      </div>

      <details className="group border-t-2 border-[#151b1e]">
        <summary className="flex h-14 cursor-pointer list-none items-center px-5 text-sm font-bold text-white">
          保证金
          <ChevronDownIcon className="ml-auto h-4 w-4 text-subtle transition group-open:rotate-180" />
        </summary>
        <div className="space-y-4 border-t border-panelBorder px-4 py-4">
          <div className="grid grid-cols-2 gap-2 text-xs">
            <div className="rounded bg-elevated px-3 py-2">
              <div className="text-subtle">名义价值</div>
              <div className="mt-1 font-mono text-white">
                {notional > 0 ? formatNumber(notional, 2) : "—"} USD
              </div>
            </div>
            <div className="rounded bg-elevated px-3 py-2">
              <div className="text-subtle">占用保证金</div>
              <div className="mt-1 font-mono text-accent">
                {marginRequired > 0
                  ? formatNumber(marginRequired, 2)
                  : "—"}{" "}
                USD
              </div>
            </div>
          </div>

          <div className="flex flex-wrap gap-1">
            {leverageOptions.map((option) => (
              <button
                key={option}
                type="button"
                onClick={() => handleLeverageChange(option)}
                className={
                  "min-w-[2.5rem] rounded px-2 py-1.5 text-[11px] font-semibold " +
                  (leverage === option
                    ? "bg-accent text-black"
                    : "bg-elevated text-muted")
                }
              >
                {option}X
              </button>
            ))}
          </div>

          <div className="flex flex-wrap gap-1">
            {SLIPPAGE_PRESETS_BPS.map((bps) => (
              <button
                key={bps}
                type="button"
                onClick={() => handleSlippageChange(bps)}
                className={
                  "rounded px-2 py-1 text-[10px] " +
                  (slippageBps === bps
                    ? "bg-amber-600/70 text-white"
                    : "bg-elevated text-muted")
                }
              >
                滑点 {(bps / 100).toFixed(1)}%
              </button>
            ))}
          </div>

          <FundingRatePanel perp={selectedPerp} compact />
        </div>
      </details>

      <details className="group border-y-2 border-[#151b1e]">
        <summary className="flex h-14 cursor-pointer list-none items-center px-5 text-sm font-bold text-white">
          资产
          <ChevronDownIcon className="ml-auto h-4 w-4 text-subtle transition group-open:rotate-180" />
        </summary>
        <div className="space-y-3 border-t border-panelBorder px-4 py-4 text-xs">
          <div className="flex justify-between text-subtle">
            <span>Dealer 链上保证金</span>
            <span className="font-mono text-white">
              {dealerCreditHuman != null
                ? formatNumber(dealerCreditHuman, 2)
                : "—"}{" "}
              USDC
            </span>
          </div>
          <div className="flex justify-between text-subtle">
            <span>标记价格</span>
            <span className="font-mono text-white">
              {markUsd > 0 ? formatNumber(markUsd, 2) : "—"}
            </span>
          </div>
          <div className="flex justify-between text-subtle">
            <span>指数价格</span>
            <span className="font-mono text-white">
              {indexPriceUsd > 0 ? formatNumber(indexPriceUsd, 2) : "—"}
            </span>
          </div>
          {openPriceSource ? (
            <div className="flex justify-between text-subtle">
              <span>价格来源</span>
              <span className="text-white">
                {formatOpenPriceSource(openPriceSource)}
              </span>
            </div>
          ) : null}
          {signLimitPriceUsd ? (
            <div className="flex justify-between text-subtle">
              <span>签名限价</span>
              <span className="font-mono text-amber-300">
                {formatNumber(Number(signLimitPriceUsd), 2)}
              </span>
            </div>
          ) : null}
          {amount.trim() ? (
            <div className="rounded bg-elevated px-3 py-2">
              <div className="flex justify-between">
                <span>深度模拟</span>
                <span>
                  {previewLoading ? "计算中…" : preview ? "已更新" : "—"}
                </span>
              </div>
              {preview ? (
                <div className="mt-2 space-y-1 font-mono text-white">
                  <div className="flex justify-between">
                    <span className="font-sans text-subtle">预估成交均价</span>
                    <span>{preview.avgFillPriceUsd || "—"}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="font-sans text-subtle">预计成交</span>
                    <span>
                      {preview.filledSize} / {preview.requestedSize}
                    </span>
                  </div>
                </div>
              ) : null}
            </div>
          ) : null}
          {creditInsufficient ? (
            <p className="text-amber-300">
              当前可用保证金不足，请先充值或领取测试 USDC。
            </p>
          ) : null}
          {leverageTooLowForChain ? (
            <p className="text-amber-300">
              所选杠杆与链上最低保证金要求不匹配。
            </p>
          ) : null}
        </div>
      </details>
    </div>
  );
}

function MarketGauge({
  label,
  buyShare,
}: {
  label: string;
  buyShare?: number;
}) {
  const clampedShare = Math.min(100, Math.max(0, buyShare ?? 50));

  return (
    <div>
      <div className="text-subtle">{label}</div>
      <div
        className="relative mx-auto mt-3 aspect-[2/1] w-24 overflow-hidden"
        aria-label={
          buyShare != null
            ? `买方占比 ${formatNumber(clampedShare, 2)}%`
            : "买卖占比暂无数据"
        }
      >
        <div
          className="absolute inset-0 rounded-t-full"
          style={{
            background:
              buyShare != null
                ? `conic-gradient(from 270deg at 50% 100%, #16c995 0deg ${clampedShare * 1.8}deg, #ff3d68 ${clampedShare * 1.8}deg 180deg, transparent 180deg 360deg)`
                : "conic-gradient(from 270deg at 50% 100%, #343a40 0deg 180deg, transparent 180deg 360deg)",
          }}
        />
        <div className="absolute bottom-0 left-1/2 h-[38px] w-[72px] -translate-x-1/2 rounded-t-full bg-[#070a0b]" />
        <div className="absolute inset-x-0 bottom-0 text-center">
          <div className="font-mono text-sm font-semibold text-buy">
            {buyShare != null ? `${formatNumber(clampedShare, 2)}%` : "—"}
          </div>
          <div className="mt-0.5 text-[10px] text-buy">
            {buyShare != null ? "买" : "暂无"}
          </div>
        </div>
      </div>
    </div>
  );
}
