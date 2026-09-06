"use client";

import { ConnectButton } from "@rainbow-me/rainbowkit";
import { useQueryClient } from "@tanstack/react-query";
import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import { isAddress, parseUnits } from "viem";
import { useAccount, useChainId, usePublicClient, useReadContract, useSwitchChain, useWaitForTransactionReceipt, useWriteContract } from "wagmi";
import { sepolia } from "wagmi/chains";
import {
  SEPOLIA_TEST_USDC,
  sepoliaTestUsdcAbi,
} from "@/lib/sepoliaTestUsdc";
import { erc20Decimals, formatTokenAmount } from "@/lib/tokenAmount";

/** 等待收据超时（毫秒）；公共 RPC 慢时避免按钮永久卡住 */
const RECEIPT_TIMEOUT_MS = 180_000;

export default function FaucetPage() {
  const qc = useQueryClient();
  const { address, isConnected } = useAccount();
  const chainId = useChainId();
  const { switchChain, isPending: switchPending } = useSwitchChain();

  const [mintTo, setMintTo] = useState("");
  const [mintAmountHuman, setMintAmountHuman] = useState("10000");
  const [txMissingOnChain, setTxMissingOnChain] = useState(false);

  const onSepolia = chainId === sepolia.id;
  const publicClient = usePublicClient({ chainId: sepolia.id });

  const {
    data: decimals,
    isPending: decimalsPending,
    isError: decimalsErr,
  } = useReadContract({
    address: SEPOLIA_TEST_USDC,
    abi: sepoliaTestUsdcAbi,
    functionName: "decimals",
    chainId: sepolia.id,
    query: { enabled: true },
  });

  const dec = erc20Decimals(decimals, 6);

  const {
    data: rawBalance,
    isPending: balancePending,
    isError: balanceErr,
    error: balanceQueryErr,
    refetch: refetchBalance,
  } = useReadContract({
    address: SEPOLIA_TEST_USDC,
    abi: sepoliaTestUsdcAbi,
    functionName: "balanceOf",
    args: address ? [address] : undefined,
    chainId: sepolia.id,
    query: { enabled: !!address && onSepolia },
  });

  const balanceFormatted = useMemo(
    () => formatTokenAmount(rawBalance, dec, 2),
    [rawBalance, dec]
  );

  useEffect(() => {
    if (address) {
      setMintTo((t) => (t.trim() === "" ? address : t));
    }
  }, [address]);

  const toOk = mintTo.trim() !== "" && isAddress(mintTo.trim() as `0x${string}`);
  let amountWei: bigint | undefined;
  try {
    amountWei = parseUnits((mintAmountHuman || "0").trim(), dec);
  } catch {
    amountWei = undefined;
  }
  const amountOk = amountWei !== undefined && amountWei > 0n;

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
    if (confirm.isSuccess) {
      setTxMissingOnChain(false);
      void qc.invalidateQueries();
      void refetchBalance();
    }
  }, [confirm.isSuccess, qc, refetchBalance]);

  useEffect(() => {
    if (!confirm.isError || !txHash || !publicClient) return;
    let cancelled = false;
    void publicClient
      .getTransaction({ hash: txHash })
      .then(() => {
        if (!cancelled) setTxMissingOnChain(false);
      })
      .catch(() => {
        if (!cancelled) setTxMissingOnChain(true);
      });
    return () => {
      cancelled = true;
    };
  }, [confirm.isError, txHash, publicClient]);

  const explorerContract = `https://sepolia.etherscan.io/address/${SEPOLIA_TEST_USDC}#code`;

  const waitingWallet = writePending && !txHash;
  const waitingReceipt = Boolean(txHash) && confirm.isPending;

  const disabledMint =
    !toOk ||
    !amountOk ||
    waitingWallet ||
    waitingReceipt;

  let primaryLabel = "mint(to, amount)";
  if (waitingWallet) primaryLabel = "请在钱包中确认…";
  else if (waitingReceipt) primaryLabel = "等待链上确认…";

  const rpcHint =
    (decimalsErr || balanceErr) &&
    !process.env.NEXT_PUBLIC_SEPOLIA_RPC_URL?.trim();

  return (
    <main className="mx-auto max-w-lg px-4 py-10">
      <div className="mb-6 flex items-center justify-between gap-4">
        <Link href="/" className="text-sm text-muted hover:text-[#B6F906]">
          ← 返回
        </Link>
        <ConnectButton chainStatus="icon" showBalance={false} />
      </div>

      <h1 className="text-xl font-semibold text-white">领取 Sepolia 测试 USDC</h1>
      <p className="mt-2 text-sm text-muted">
        合约{" "}
        <a
          href={explorerContract}
          target="_blank"
          rel="noreferrer"
          className="text-[#B6F906] underline-offset-2 hover:underline"
        >
          {SEPOLIA_TEST_USDC}
        </a>
      </p>

      {!isConnected ? (
        <p className="mt-6 text-sm text-muted">请先连接钱包。</p>
      ) : !onSepolia ? (
        <div className="mt-6 space-y-3">
          <p className="text-sm text-amber-200">当前网络不是 Sepolia，请切换。</p>
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
        <div className="mt-6 space-y-4 rounded-xl border border-panelBorder bg-white/5 p-5">
          <div>
            <span className="text-muted">当前钱包余额 </span>
            <span className="font-mono text-white">{balanceFormatted} USDC</span>
            {balancePending && (
              <span className="ml-2 text-xs text-muted">读取中…</span>
            )}
          </div>
          {(decimalsErr || balanceErr) && (
            <p className="text-sm text-amber-200">
              链上读取失败（多为 RPC 限速或网络问题）。
              {rpcHint ? (
                <>
                  {" "}
                  请在项目根目录配置{" "}
                  <code className="text-muted">NEXT_PUBLIC_SEPOLIA_RPC_URL</code>{" "}
                  指向 Alchemy / Infura 等 Sepolia 节点后重启 dev。
                </>
              ) : null}
              {balanceQueryErr?.message ? (
                <span className="mt-1 block text-xs text-muted">{balanceQueryErr.message}</span>
              ) : null}
            </p>
          )}

          <div className="space-y-1">
            <label className="text-xs text-muted">接收地址 to</label>
            <input
              type="text"
              value={mintTo}
              onChange={(e) => setMintTo(e.target.value.trim())}
              placeholder="0x…"
              className="w-full rounded-lg border border-panelBorder bg-elevated px-3 py-2 font-mono text-sm text-white placeholder:text-muted"
            />
          </div>

          <div className="space-y-1">
            <label className="text-xs text-muted">
              数量 amount（人类可读，{decimalsPending ? "…" : dec} 位小数）
            </label>
            <input
              type="text"
              inputMode="decimal"
              value={mintAmountHuman}
              onChange={(e) => setMintAmountHuman(e.target.value)}
              placeholder="10000"
              className="w-full rounded-lg border border-panelBorder bg-elevated px-3 py-2 font-mono text-sm text-white placeholder:text-muted"
            />
          </div>

          <button
            type="button"
            disabled={disabledMint}
            onClick={() => {
              if (!toOk || !amountOk || amountWei === undefined) return;
              resetWrite();
              setTxMissingOnChain(false);
              writeContract({
                address: SEPOLIA_TEST_USDC,
                abi: sepoliaTestUsdcAbi,
                functionName: "mint",
                args: [mintTo.trim() as `0x${string}`, amountWei],
                chainId: sepolia.id,
              });
            }}
            className="w-full rounded-lg bg-[#B6F906] px-4 py-3 text-sm font-semibold text-black disabled:opacity-45"
          >
            {primaryLabel}
          </button>

          {(waitingWallet || waitingReceipt) && (
            <button
              type="button"
              className="w-full rounded-lg border border-panelBorder bg-transparent px-4 py-2 text-sm text-muted hover:bg-panelBorder"
              onClick={() => {
                resetWrite();
                void qc.invalidateQueries();
              }}
            >
              卡住？重置状态
            </button>
          )}

          <p className="text-xs text-muted">
            调用{" "}
            <code className="text-muted">function mint(address to, uint256 amount) external</code>
          </p>
          {waitingWallet && (
            <p className="text-xs text-muted">
              若浏览器未弹出钱包窗口，请检查扩展图标或是否被浏览器拦截弹窗。
            </p>
          )}

          {writeErr && (
            <p className="text-sm text-red-400">{writeErr.message}</p>
          )}
          {confirm.isError && (
            <p className="text-sm text-red-400">
              {txMissingOnChain
                ? "交易未出现在 Sepolia 链上（可能被钱包拒绝、网络/RPC 异常或交易被丢弃）。请确认钱包已签名并切换到 Sepolia，然后重试 mint。"
                : `等待确认超时或失败：${confirm.error?.message ?? "unknown"}`}
              {txHash ? (
                <>
                  {" "}
                  可自行到{" "}
                  <a
                    href={`https://sepolia.etherscan.io/tx/${txHash}`}
                    target="_blank"
                    rel="noreferrer"
                    className="text-[#B6F906] underline"
                  >
                    Etherscan
                  </a>{" "}
                  查看；若页面显示 Not found，说明交易并未成功广播。
                </>
              ) : null}
            </p>
          )}
          {confirm.isSuccess && txHash && (
            <p className="text-sm text-emerald-400/90">
              已 mint {mintAmountHuman.trim()} USDC。刷新后余额应增加{" "}
              {mintAmountHuman.trim()}（6 位小数）。
            </p>
          )}
          {txHash && (
            <p className="text-xs">
              <a
                href={`https://sepolia.etherscan.io/tx/${txHash}`}
                target="_blank"
                rel="noreferrer"
                className="text-[#B6F906] underline-offset-2 hover:underline"
              >
                查看交易
              </a>
            </p>
          )}
        </div>
      )}
    </main>
  );
}
