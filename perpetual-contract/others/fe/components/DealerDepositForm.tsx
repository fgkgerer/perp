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
  metaNodeDealerAbi,
  resolveDealerAddress,
} from "@/lib/metanodeDealer";
import {
  resolveUsdcAddress,
  sepoliaTestUsdcAbi,
} from "@/lib/sepoliaTestUsdc";
import { erc20Decimals, formatTokenAmount } from "@/lib/tokenAmount";

const RECEIPT_TIMEOUT_MS = 180_000;

type Props = {
  onDeposited?: (txHash: `0x${string}`) => void;
  compact?: boolean;
};

type Step = "idle" | "approving" | "depositing";

export default function DealerDepositForm({ onDeposited, compact }: Props) {
  const usdcAddress = resolveUsdcAddress();
  const dealerAddress = resolveDealerAddress();

  const { address, isConnected } = useAccount();
  const chainId = useChainId();
  const { switchChain, isPending: switchPending } = useSwitchChain();

  const [amountHuman, setAmountHuman] = useState("");
  const [step, setStep] = useState<Step>("idle");
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

  const {
    data: rawAllowance,
    refetch: refetchAllowance,
  } = useReadContract({
    address: usdcAddress,
    abi: sepoliaTestUsdcAbi,
    functionName: "allowance",
    args: address ? [address, dealerAddress] : undefined,
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

  const needsApprove =
    amountOk &&
    amountWei !== undefined &&
    (rawAllowance == null || rawAllowance < amountWei);

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

    const run = async () => {
      if (step === "approving" && amountOk && amountWei !== undefined && address) {
        await refetchAllowance();
        resetWrite();
        setStep("depositing");
        writeContract({
          address: dealerAddress,
          abi: metaNodeDealerAbi,
          functionName: "deposit",
          args: [amountWei, 0n, address],
          chainId: sepolia.id,
        });
        return;
      }
      if (step === "depositing") {
        await refetchBalance();
        await refetchAllowance();
        setAmountHuman("");
        setStep("idle");
        onDeposited?.(txHash);
      }
    };
    void run();
  }, [
    confirm.isSuccess,
    txHash,
    step,
    amountOk,
    amountWei,
    address,
    dealerAddress,
    refetchAllowance,
    refetchBalance,
    resetWrite,
    writeContract,
    onDeposited,
  ]);

  const waitingWallet = writePending && !txHash;
  const waitingReceipt = Boolean(txHash) && confirm.isPending;

  const disabled =
    !isConnected ||
    !onSepolia ||
    !amountOk ||
    waitingWallet ||
    waitingReceipt ||
    step === "depositing";

  let btnLabel = "存入链上保证金";
  if (step === "approving" || (needsApprove && step === "idle")) {
    btnLabel = waitingWallet
      ? "请在钱包中确认授权…"
      : waitingReceipt
        ? "等待授权确认…"
        : "授权 USDC 并充值";
  } else if (waitingWallet) {
    btnLabel = "请在钱包中确认充值…";
  } else if (waitingReceipt) {
    btnLabel = "等待链上确认…";
  }

  const txUrl = txHash
    ? `https://sepolia.etherscan.io/tx/${txHash}`
    : null;

  const shellClass = compact
    ? "space-y-4"
    : "mt-4 space-y-4 rounded-lg border border-panelBorder bg-elevated p-4";

  return (
    <div className={shellClass}>
      <div className="text-xs text-muted">
        Dealer 合约{" "}
        <code className="break-all text-muted">{dealerAddress}</code>
      </div>

      {!isConnected ? (
        <p className="text-sm text-muted">连接钱包后可存入链上保证金。</p>
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
            <label className="text-xs text-muted">保证金数量</label>
            <div className="flex gap-2">
              <input
                type="text"
                inputMode="decimal"
                value={amountHuman}
                onChange={(e) => setAmountHuman(e.target.value)}
                placeholder="10000"
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
              if (!amountOk || amountWei === undefined || !address) return;
              if (!isAddress(dealerAddress)) return;
              resetWrite();
              if (needsApprove) {
                setStep("approving");
                writeContract({
                  address: usdcAddress,
                  abi: sepoliaTestUsdcAbi,
                  functionName: "approve",
                  args: [dealerAddress, amountWei],
                  chainId: sepolia.id,
                });
              } else {
                setStep("depositing");
                writeContract({
                  address: dealerAddress,
                  abi: metaNodeDealerAbi,
                  functionName: "deposit",
                  args: [amountWei, 0n, address],
                  chainId: sepolia.id,
                });
              }
            }}
            className="w-full rounded-lg bg-emerald-500 px-4 py-3 text-sm font-semibold text-black disabled:opacity-45"
          >
            {btnLabel}
          </button>

          <p className="text-xs leading-relaxed text-muted">
            调用{" "}
            <code className="text-muted">MetaNodeDealer.deposit</code>
            ，USDC 进入 Dealer 合约，用于永续开仓/成交。与下方「托管充值」不同。
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
          {confirm.isSuccess && txUrl && step === "idle" ? (
            <p className="text-sm text-emerald-400/90">
              链上保证金已存入。{" "}
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
