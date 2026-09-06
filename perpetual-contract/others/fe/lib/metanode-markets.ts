/** Sepolia MetaNode 永续市场（与 README / metanode.yaml 一致） */

export type MetaNodeMarket = {
  name: string;
  address: `0x${string}`;
  symbol: string;
};

export const SEPOLIA_METANODE_MARKETS: MetaNodeMarket[] = [
  {
    name: "BTC-PERP",
    address: "0x11Aae1f92Ff10bfbb205971e060CF6d9D917723b",
    symbol: "BTC",
  },
  {
    name: "ETH-PERP",
    address: "0x98456DCbcEfea550293727A7E2DfD45De92740c0",
    symbol: "ETH",
  },
];

export const SEPOLIA_CHAIN_ID = 11155111;

/**
 * 标记价 / 系统开仓价：支持 API 返回的 2 位小数字符串，或链上 1e6 / 1e18 整数串。
 */
export function markPriceToUsd(raw: string | undefined | null): number {
  if (!raw) return 0;
  const s = raw.trim();
  if (s.includes(".")) {
    const n = Number.parseFloat(s);
    return Number.isFinite(n) ? n : 0;
  }
  try {
    const bi = BigInt(s);
    if (bi >= 10n ** 15n) return Number(bi) / 1e18;
    return Number(bi) / 1e6;
  } catch {
    return 0;
  }
}

export function paperToSize(raw: string): number {
  try {
    return Number(BigInt(raw)) / 1e18;
  } catch {
    return 0;
  }
}
