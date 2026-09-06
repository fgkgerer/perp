"use client";

import Link from "next/link";
import { useCallback, useEffect, useMemo, useState } from "react";
import { ConnectButton } from "@rainbow-me/rainbowkit";
import { formatUnits } from "viem";
import { useAccount } from "wagmi";
import {
  fetchMetanodeBalance,
  fetchMetanodeDeposits,
  METANODE_API_BASE,
  type AccountBalanceDTO,
  type DepositRecordDTO,
} from "@/lib/metanode-api";
import { useMetanodeAuth } from "@/lib/useMetanodeAuth";
import { resolveTreasuryAddress } from "@/lib/sepoliaTestUsdc";
import UsdcDepositModal from "@/components/UsdcDepositModal";
import DealerDepositModal from "@/components/DealerDepositModal";

const USDC_DECIMALS = 6;
const DEPOSIT_POLL_MS = 5_000;
const DEPOSIT_POLL_MAX = 36;

function formatCredit(raw: string): string {
  try {
    const v = BigInt(raw.trim() || "0");
    return formatUnits(v, USDC_DECIMALS);
  } catch {
    return raw;
  }
}

function txExplorerUrl(txHash: string): string {
  const tpl =
    typeof process.env.NEXT_PUBLIC_EXPLORER_TX === "string" &&
    process.env.NEXT_PUBLIC_EXPLORER_TX.trim() !== ""
      ? process.env.NEXT_PUBLIC_EXPLORER_TX.trim()
      : "https://sepolia.etherscan.io/tx/{hash}";
  if (tpl.includes("{hash}")) {
    return tpl.replace("{hash}", txHash);
  }
  const base = tpl.replace(/\/$/, "");
  return `${base}/${txHash}`;
}

function formatTime(sec: number): string {
  if (!Number.isFinite(sec) || sec <= 0) return "—";
  return new Date(sec * 1000).toLocaleString("zh-CN");
}

type Props = {
  onBack: () => void;
};

export default function UserProfile({ onBack }: Props) {
  const { address, isConnected } = useAccount();
  const [balance, setBalance] = useState<AccountBalanceDTO | null>(null);
  const [deposits, setDeposits] = useState<DepositRecordDTO[]>([]);
  const [depTotal, setDepTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const [depositOpen, setDepositOpen] = useState(false);
  const [dealerDepositOpen, setDealerDepositOpen] = useState(false);
  const [syncingDeposit, setSyncingDeposit] = useState(false);
  const [syncTxHash, setSyncTxHash] = useState<string | null>(null);
  const [syncAttempt, setSyncAttempt] = useState(0);

  const treasury = resolveTreasuryAddress();

  const { sessionOk, authPending } = useMetanodeAuth(address);

  const applyAccountData = useCallback(
    (b: Awaited<ReturnType<typeof fetchMetanodeBalance>>, d: Awaited<ReturnType<typeof fetchMetanodeDeposits>>) => {
      if (b.code !== 0) {
        setErr(b.message || `余额接口 code=${b.code}`);
        setBalance(null);
      } else {
        setBalance(b.balance);
      }
      if (d.code !== 0) {
        setErr((prev) => prev ?? (d.message || `充值记录 code=${d.code}`));
        setDeposits([]);
        setDepTotal(0);
      } else {
        setDeposits(d.deposits);
        setDepTotal(Number(d.total));
      }
    },
    []
  );

  const load = useCallback(
    async (opts?: { silent?: boolean }) => {
      if (!address || !isConnected) return null;
      if (!opts?.silent) {
        setLoading(true);
        setErr(null);
      }
      try {
        const trader = address.trim();
        const [b, d] = await Promise.all([
          fetchMetanodeBalance(trader),
          fetchMetanodeDeposits(trader, 1, 20),
        ]);
        applyAccountData(b, d);
        return { b, d };
      } catch (e) {
        if (!opts?.silent) {
          setErr(e instanceof Error ? e.message : String(e));
          setBalance(null);
          setDeposits([]);
        }
        return null;
      } finally {
        if (!opts?.silent) setLoading(false);
      }
    },
    [address, isConnected, applyAccountData]
  );

  const startDepositSync = useCallback((txHash: `0x${string}`) => {
    setSyncTxHash(txHash);
    setSyncAttempt(0);
    setSyncingDeposit(true);
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    if (!syncingDeposit || !syncTxHash || !address || !isConnected) return;

    let cancelled = false;
    let attempts = 0;

    const tick = async () => {
      if (cancelled) return;
      attempts += 1;
      setSyncAttempt(attempts);
      const result = await load({ silent: true });
      if (cancelled || !result) return;

      const hashLower = syncTxHash.toLowerCase();
      const found = result.d.deposits.some(
        (row) => row.txHash.toLowerCase() === hashLower
      );

      if (found || attempts >= DEPOSIT_POLL_MAX) {
        setSyncingDeposit(false);
        setSyncTxHash(null);
        setSyncAttempt(0);
      }
    };

    void tick();
    const id = window.setInterval(() => void tick(), DEPOSIT_POLL_MS);
    return () => {
      cancelled = true;
      window.clearInterval(id);
    };
  }, [syncingDeposit, syncTxHash, address, isConnected, load]);

  const copyTreasury = async () => {
    if (!treasury) return;
    try {
      await navigator.clipboard.writeText(treasury);
    } catch {
      /* ignore */
    }
  };

  return (
    <div className="min-h-screen bg-page text-foreground">
      <header className="flex flex-wrap items-center justify-between gap-3 border-b border-panelBorder px-4 py-3 md:px-6">
        <div className="flex flex-wrap items-center gap-3">
          <button
            type="button"
            onClick={onBack}
            className="rounded-lg bg-elevated px-3 py-2 text-sm font-semibold text-foreground hover:bg-panelBorder"
          >
            返回交易
          </button>
          <Link
            href="/positions"
            className="rounded-lg border border-panelBorder px-3 py-2 text-sm font-semibold text-foreground hover:bg-panelBorder"
          >
            持仓
          </Link>
          <h1 className="text-lg font-semibold text-white">我的账户</h1>
        </div>
        <ConnectButton />
      </header>

      <main className="mx-auto max-w-3xl space-y-6 px-4 py-6 md:px-6">
        {!isConnected || !address ? (
          <p className="rounded-xl border border-panelBorder bg-panel p-4 text-muted">
            请先连接钱包；连接后将自动完成 MetaNode 登录（签名），然后即可查询余额与充值记录。
          </p>
        ) : (
          <>
            <section className="rounded-xl border border-panelBorder bg-panel p-4">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <h2 className="text-sm font-semibold text-white">钱包地址</h2>
                {!sessionOk ? (
                  <span className="text-xs text-amber-400">
                    {authPending
                      ? "MetaNode 登录中，请在钱包完成登录签名…"
                      : `会话未就绪：请刷新页面并在钱包完成 MetaNode 登录签名（API ${METANODE_API_BASE}）`}
                  </span>
                ) : (
                  <span className="text-xs text-emerald-400/90">已登录</span>
                )}
              </div>
              <p className="mt-2 break-all font-mono text-xs text-muted md:text-sm">
                {address}
              </p>
            </section>

            <section className="rounded-xl border border-panelBorder bg-panel p-4">
              <h2 className="text-sm font-semibold text-white">两种充值方式对比</h2>
              <div className="mt-3 overflow-x-auto">
                <table className="min-w-[520px] w-full text-left text-xs text-muted">
                  <thead>
                    <tr className="border-b border-panelBorder text-muted">
                      <th className="py-2 pr-3 font-medium"> </th>
                      <th className="py-2 pr-3 font-medium text-emerald-400">
                        链上保证金（Dealer）
                      </th>
                      <th className="py-2 font-medium text-[#B6F906]">
                        托管充值（Treasury）
                      </th>
                    </tr>
                  </thead>
                  <tbody className="text-muted">
                    <tr className="border-b border-panelBorder">
                      <td className="py-2 pr-3 text-muted">操作</td>
                      <td className="py-2 pr-3">approve + Dealer.deposit</td>
                      <td className="py-2">USDC transfer 到托管地址</td>
                    </tr>
                    <tr className="border-b border-panelBorder">
                      <td className="py-2 pr-3 text-muted">资金去向</td>
                      <td className="py-2 pr-3">MetaNodeDealer 合约</td>
                      <td className="py-2">Treasury 收款地址</td>
                    </tr>
                    <tr className="border-b border-panelBorder">
                      <td className="py-2 pr-3 text-muted">记账</td>
                      <td className="py-2 pr-3">链上 getCreditOf</td>
                      <td className="py-2">后端扫链 → ledger_balances</td>
                    </tr>
                    <tr>
                      <td className="py-2 pr-3 text-muted">能否成交</td>
                      <td className="py-2 pr-3 text-emerald-400">✓ 永续开仓必需</td>
                      <td className="py-2 text-amber-300">✗ 不能单独成交</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </section>

            <section className="rounded-xl border border-emerald-500/20 bg-panel p-4">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <h2 className="text-sm font-semibold text-white">链上保证金（Dealer）</h2>
                <button
                  type="button"
                  onClick={() => setDealerDepositOpen(true)}
                  className="rounded-lg bg-emerald-500 px-3 py-1.5 text-xs font-semibold text-black hover:opacity-90"
                >
                  存入保证金
                </button>
              </div>
              <p className="mt-3 text-sm leading-relaxed text-muted">
                调用{" "}
                <strong className="text-foreground">MetaNodeDealer.deposit</strong>
                ，USDC 进入 Dealer 合约，是永续开仓/成交所必需的链上保证金。
                下方「托管充值」仅计入链下 ledger，不能单独用于成交。
              </p>
            </section>

            <section className="rounded-xl border border-panelBorder bg-panel p-4">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <h2 className="text-sm font-semibold text-white">托管充值（Treasury）</h2>
                <div className="flex flex-wrap gap-2">
                  <button
                    type="button"
                    onClick={() => setDepositOpen(true)}
                    disabled={!treasury}
                    className="rounded-lg bg-[#B6F906] px-3 py-1.5 text-xs font-semibold text-black hover:opacity-90 disabled:opacity-45"
                  >
                    充值 USDC
                  </button>
                  <button
                    type="button"
                    disabled={loading}
                    onClick={() => void load()}
                    className="rounded-lg bg-elevated px-3 py-1.5 text-xs font-semibold text-foreground hover:bg-panelBorder disabled:opacity-50"
                  >
                    {loading ? "刷新中…" : "刷新余额"}
                  </button>
                </div>
              </div>
              {syncingDeposit ? (
                <p className="mt-3 rounded-lg border border-[#B6F906]/30 bg-[#B6F906]/10 px-3 py-2 text-xs text-[#B6F906]">
                  链上已成功，正在等待后端入账…（第 {syncAttempt}/{DEPOSIT_POLL_MAX}{" "}
                  次，每 {DEPOSIT_POLL_MS / 1000}s 自动刷新）
                  {syncTxHash ? (
                    <>
                      {" "}
                      <a
                        href={txExplorerUrl(syncTxHash)}
                        target="_blank"
                        rel="noreferrer"
                        className="underline"
                      >
                        交易
                      </a>
                    </>
                  ) : null}
                </p>
              ) : null}
              <p className="mt-3 text-sm leading-relaxed text-muted">
                向平台托管地址转入{" "}
                <strong className="text-foreground">USDC</strong>（Sepolia）。
                链上确认后由后端扫链入账 MetaNode 链下余额（ledger），
                <strong className="text-amber-200/90"> 不等同于 Dealer 保证金</strong>。
              </p>
              {treasury ? (
                <div className="mt-3 flex flex-col gap-2 sm:flex-row sm:items-center">
                  <code className="block flex-1 break-all rounded-lg bg-elevated px-3 py-2 font-mono text-xs text-muted">
                    {treasury}
                  </code>
                  <button
                    type="button"
                    onClick={() => void copyTreasury()}
                    className="shrink-0 rounded-lg bg-elevated px-3 py-2 text-xs font-semibold text-foreground hover:bg-panelBorder"
                  >
                    复制地址
                  </button>
                </div>
              ) : (
                <p className="mt-3 text-sm text-amber-400">
                  未配置托管地址{" "}
                  <code className="text-xs">NEXT_PUBLIC_USDC_TREASURY_ADDRESS</code>
                </p>
              )}
              <p className="mt-3 text-xs text-muted">
                测试币可到{" "}
                <Link href="/faucet" className="text-[#B6F906] underline">
                  水龙头页
                </Link>
                领取。
              </p>
            </section>

            <section className="rounded-xl border border-panelBorder bg-panel p-4">
              <h2 className="text-sm font-semibold text-white">账户余额</h2>
              <p className="mt-1 text-xs text-muted">
                主资产拆分为链上 Dealer 保证金与 Treasury 链下 ledger；合计为接口 primaryCredit。
              </p>
              {err ? (
                <p className="mt-3 text-sm text-red-400">{err}</p>
              ) : balance ? (
                <dl className="mt-4 grid gap-3 text-sm sm:grid-cols-2">
                  <div className="sm:col-span-2 rounded-lg border border-emerald-500/20 bg-emerald-500/5 px-3 py-2">
                    <dt className="text-emerald-400/90">链上 Dealer 保证金</dt>
                    <dd className="font-mono text-base text-white">
                      {formatCredit(balance.onChainPrimaryCredit ?? "0")} USDC
                    </dd>
                  </div>
                  <div className="sm:col-span-2 rounded-lg border border-[#B6F906]/20 bg-[#B6F906]/5 px-3 py-2">
                    <dt className="text-[#B6F906]/90">Treasury 链下 ledger</dt>
                    <dd className="font-mono text-base text-white">
                      {formatCredit(balance.ledgerPrimaryBalance ?? "0")} USDC
                    </dd>
                  </div>
                  <div className="sm:col-span-2 border-t border-panelBorder pt-3">
                    <dt className="text-muted">主资产合计</dt>
                    <dd className="font-mono text-lg text-white">
                      {formatCredit(balance.primaryCredit)} USDC
                    </dd>
                  </div>
                  <div>
                    <dt className="text-muted">次级资产余额</dt>
                    <dd className="font-mono text-base text-white">
                      {formatCredit(balance.secondaryCredit)}
                    </dd>
                  </div>
                  <div>
                    <dt className="text-muted">待取款（主）</dt>
                    <dd className="font-mono text-muted">
                      {formatCredit(balance.pendingPrimaryWithdraw)}
                    </dd>
                  </div>
                  <div>
                    <dt className="text-muted">待取款（次）</dt>
                    <dd className="font-mono text-muted">
                      {formatCredit(balance.pendingSecondaryWithdraw)}
                    </dd>
                  </div>
                  <div className="sm:col-span-2">
                    <dt className="text-muted">可执行取款时间</dt>
                    <dd className="text-muted">
                      {formatTime(balance.executionTimestamp)}
                    </dd>
                  </div>
                </dl>
              ) : (
                <p className="mt-3 text-sm text-muted">
                  {loading ? "加载中…" : "暂无数据"}
                </p>
              )}
            </section>

            <section className="rounded-xl border border-panelBorder bg-panel p-4">
              <div className="flex flex-wrap items-baseline justify-between gap-2">
                <h2 className="text-sm font-semibold text-white">Treasury 充值记录</h2>
                {depTotal > 0 ? (
                  <span className="text-xs text-muted">
                    共 {depTotal} 条（本页 {deposits.length} 条）
                  </span>
                ) : null}
              </div>
              {deposits.length === 0 && !loading ? (
                <p className="mt-3 text-sm text-muted">暂无充值记录</p>
              ) : (
                <ul className="mt-4 divide-y divide-panelBorder">
                  {deposits.map((row) => (
                    <li key={row.txHash} className="py-3 first:pt-0">
                      <div className="flex flex-wrap items-center gap-2">
                        <a
                          href={txExplorerUrl(row.txHash)}
                          target="_blank"
                          rel="noreferrer"
                          className="break-all font-mono text-xs text-[#B6F906] underline"
                        >
                          {row.txHash.slice(0, 10)}…{row.txHash.slice(-8)}
                        </a>
                        <span className="text-xs text-muted">
                          {formatTime(row.createTime)}
                        </span>
                      </div>
                      <div className="mt-1 text-xs text-muted">
                        主 +{formatCredit(row.primaryAmount)} · 次 +
                        {formatCredit(row.secondaryAmount)} · 区块{" "}
                        {String(row.blockNumber)}
                      </div>
                    </li>
                  ))}
                </ul>
              )}
            </section>

          </>
        )}
      </main>

      <UsdcDepositModal
        open={depositOpen}
        onClose={() => setDepositOpen(false)}
        onDeposited={startDepositSync}
      />
      <DealerDepositModal
        open={dealerDepositOpen}
        onClose={() => setDealerDepositOpen(false)}
        onDeposited={() => void load()}
      />
    </div>
  );
}
