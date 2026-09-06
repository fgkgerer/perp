"use client";

import { useEffect, useMemo, useState } from "react";
import { formatUnits, isAddress, parseUnits } from "viem";
import {
  useAccount,
  useChainId,
  useReadContract,
  useSwitchChain,
  useWaitForTransactionReceipt,
  useWriteContract,
} from "wagmi";
import { sepolia } from "wagmi/chains";
import {
  resolveTreasuryAddress,
  resolveUsdcAddress,
  sepoliaTestUsdcAbi,
} from "@/lib/sepoliaTestUsdc";
import { erc20Decimals, formatTokenAmount } from "@/lib/tokenAmount";

const RECEIPT_TIMEOUT_MS = 180_000;
const POLL_INTERVAL_MS = 5_000;

type Props = {
  /** 链上 transfer 确认成功后回调（含 txHash） */
  onDeposited?: (txHash: `0x${string}`) => void;
  /** 弹窗内使用时去掉外层卡片样式 */
  compact?: boolean;
};

export default function UsdcDepositForm({ onDeposited, compact }: Props) {
  const usdcAddress = resolveUsdcAddress();
  const treasury = resolveTreasuryAddress();

  const { address, isConnected } = useAccount();
  const chainId = useChainId();
  const { switchChain, isPending: switchPending } = useSwitchChain();

  const [amountHuman, setAmountHuman] = useState("");
  const onSepolia = chainId === sepolia.id;

  const { data: decimals } = useReadContract({
    address: usdcAddress,
    abi: sepoliaTestUsdcAbi,
    functionName: "decimals",
    chainId: sepolia.id,
  });
  const dec = erc20Decimals(decimals, 6);

  const {
    data: rawBalance,
    isPending: balancePending,
    refetch: refetchBalance,
  } = useReadContract({
    address: usdcAddress,
    abi: sepoliaTestUsdcAbi,
    functionName: "balanceOf",
    args: address ? [address] : undefined,
    chainId: sepolia.id,
    query: { enabled: Boolean(address && onSepolia) },
  });

  const walletBalance = useMemo(
    () => formatTokenAmount(rawBalance, dec, 2),
    [rawBalance, dec]
  );

  let amountWei: bigint | undefined;
  try {
    amountWei = parseUnits((amountHuman || "0").trim(), dec);
  } catch {
    amountWei = undefined;
  }
  const amountOk =
    amountWei !== undefined &&
    amountWei > 0n &&
    (rawBalance == null || amountWei <= rawBalance);

  const {
    writeContract,
    data: txHash,
    isPending: writePending,
    error: writeErr,
    reset: resetWrite,
  } = useWriteContract();

  const confirm = useWaitForTransactionReceipt({
    hash: txHash,
    chainId: sepolia.id,
    timeout: RECEIPT_TIMEOUT_MS,
    query: {
      enabled: Boolean(txHash),
      retry: true,
      retryDelay: 2000,
    },
  });

  useEffect(() => {
    if (!confirm.isSuccess || !txHash) return;
    void refetchBalance();
    setAmountHuman("");
    onDeposited?.(txHash);
  }, [confirm.isSuccess, txHash, refetchBalance, onDeposited]);

  const waitingWallet = writePending && !txHash;
  const waitingReceipt = Boolean(txHash) && confirm.isPending;

  const disabled =
    !treasury ||
    !isConnected ||
    !onSepolia ||
    !amountOk ||
    waitingWallet ||
    waitingReceipt;

  let btnLabel = "充值 USDC 到托管账户";
  if (waitingWallet) btnLabel = "请在钱包中确认…";
  else if (waitingReceipt) btnLabel = "等待链上确认…";

  const txUrl = txHash
    ? `https://sepolia.etherscan.io/tx/${txHash}`
    : null;

  if (!treasury) {
    return (
      <p className="mt-3 text-sm text-amber-400">
        未配置{" "}
        <code className="text-xs">NEXT_PUBLIC_USDC_TREASURY_ADDRESS</code>
        （需与后端 UsdcTreasuryAddress 一致）。
      </p>
    );
  }

  const shellClass = compact
    ? "space-y-4"
    : "mt-4 space-y-4 rounded-lg border border-panelBorder bg-elevated p-4";

  return (
    <div className={shellClass}>
      <div className="text-xs text-muted">
        USDC 合约{" "}
        <code className="text-muted">{usdcAddress}</code>
      </div>
      <div className="text-xs text-muted">
        托管收款{" "}
        <code className="break-all text-muted">{treasury}</code>
      </div>

      {!isConnected ? (
        <p className="text-sm text-muted">连接钱包后可充值。</p>
      ) : !onSepolia ? (
        <div className="space-y-2">
          <p className="text-sm text-amber-200">请切换到 Sepolia 网络。</p>
          <button
            type="button"
            disabled={switchPending}
            onClick={() => switchChain({ chainId: sepolia.id })}
            className="rounded-lg bg-[#B6F906] px-4 py-2 text-sm font-semibold text-black disabled:opacity-50"
          >
            切换到 Sepolia
          </button>
        </div>
      ) : (
        <>
          <div className="text-sm">
            <span className="text-muted">钱包 USDC 余额 </span>
            <span className="font-mono text-white">
              {balancePending ? "读取中…" : `${walletBalance} USDC`}
            </span>
          </div>

          <div className="space-y-1">
            <label className="text-xs text-muted">充值数量</label>
            <div className="flex gap-2">
              <input
                type="text"
                inputMode="decimal"
                value={amountHuman}
                onChange={(e) => setAmountHuman(e.target.value)}
                placeholder="100"
                className="min-w-0 flex-1 rounded-lg border border-panelBorder bg-elevated px-3 py-2 font-mono text-sm text-white placeholder:text-muted"
              />
              <button
                type="button"
                disabled={rawBalance == null || rawBalance === 0n}
                onClick={() => {
                  if (rawBalance != null) {
                    setAmountHuman(formatUnits(rawBalance, dec));
                  }
                }}
                className="shrink-0 rounded-lg border border-panelBorder px-3 py-2 text-xs text-muted hover:bg-panelBorder"
              >
                最大
              </button>
            </div>
          </div>

          <button
            type="button"
            disabled={disabled}
            onClick={() => {
              if (!amountOk || amountWei === undefined || !isAddress(treasury)) {
                return;
              }
              resetWrite();
              writeContract({
                address: usdcAddress,
                abi: sepoliaTestUsdcAbi,
                functionName: "transfer",
                args: [treasury, amountWei],
                chainId: sepolia.id,
              });
            }}
            className="w-full rounded-lg bg-[#B6F906] px-4 py-3 text-sm font-semibold text-black disabled:opacity-45"
          >
            {btnLabel}
          </button>

          <p className="text-xs leading-relaxed text-muted">
            链上确认后将自动轮询 MetaNode 入账状态（约每{" "}
            {POLL_INTERVAL_MS / 1000} 秒），无需手动刷新。
          </p>

          {writeErr ? (
            <p className="text-sm text-red-400">{writeErr.message}</p>
          ) : null}
          {confirm.isError ? (
            <p className="text-sm text-red-400">
              {confirm.error?.message ?? "确认超时"}
              {txUrl ? (
                <>
                  {" "}
                  <a
                    href={txUrl}
                    target="_blank"
                    rel="noreferrer"
                    className="text-[#B6F906] underline"
                  >
                    查看交易
                  </a>
                </>
              ) : null}
            </p>
          ) : null}
          {confirm.isSuccess && txUrl ? (
            <p className="text-sm text-emerald-400/90">
              链上转账成功。{" "}
              <a
                href={txUrl}
                target="_blank"
                rel="noreferrer"
                className="text-[#B6F906] underline"
              >
                查看交易
              </a>
            </p>
          ) : null}
        </>
      )}
    </div>
  );
}
