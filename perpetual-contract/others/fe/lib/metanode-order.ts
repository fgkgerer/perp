import { parseUnits, type Address, type Hex } from "viem";
import { SEPOLIA_CHAIN_ID } from "@/lib/metanode-markets";
import { resolveDealerAddress } from "@/lib/metanodeDealer";

/** 与 TradingInit.buildOrder / ScenarioTradingTest 一致 */
export const MAKER_FEE_RATE = 100_000_000_000_000n; // 1e14 = 0.01%
export const TAKER_FEE_RATE = 500_000_000_000_000n; // 5e14 = 0.05%
export const ORDER_TTL_SEC = 3600;

export const METANODE_ORDER_TYPES = {
  Order: [
    { name: "perp", type: "address" },
    { name: "signer", type: "address" },
    { name: "paperAmount", type: "int128" },
    { name: "creditAmount", type: "int128" },
    { name: "info", type: "bytes32" },
  ],
} as const;

export type OrderSide = "long" | "short";

export type BuildOrderInput = {
  perp: Address;
  signer: Address;
  side: OrderSide;
  /** 标的数量，如 1 BTC */
  size: string;
  /** USDC 限价，如 30000 */
  priceUsd: string;
  expiration?: number;
  nonce?: number;
};

export type MetaNodeOrderPayload = {
  perp: Address;
  signer: Address;
  paperAmount: bigint;
  creditAmount: bigint;
  info: Hex;
  makerFeeRate: string;
  takerFeeRate: string;
  expiration: number;
  nonce: number;
};

/** 与 backend chain.PackOrderInfo 一致 */
export function packOrderInfo(
  makerFee: bigint,
  takerFee: bigint,
  expiration: bigint,
  nonce: bigint
): Hex {
  const info =
    (makerFee << 192n) |
    (takerFee << 128n) |
    (expiration << 64n) |
    nonce;
  let hex = info.toString(16);
  if (hex.length > 64) hex = hex.slice(-64);
  return `0x${hex.padStart(64, "0")}` as Hex;
}

/** paper 1e18、credit 6 位 USDC，多空符号与 ScenarioTradingTest 一致 */
export function buildOrderAmounts(
  side: OrderSide,
  sizeHuman: string,
  priceUsdHuman: string
): { paperAmount: bigint; creditAmount: bigint } {
  const size = parseUnits(sizeHuman.trim() || "0", 18);
  if (size <= 0n) {
    throw new Error("数量须大于 0");
  }
  const price6 = parseUnits(priceUsdHuman.trim() || "0", 6);
  if (price6 <= 0n) {
    throw new Error("价格须大于 0");
  }
  const notional = (size * price6) / 10n ** 18n;
  if (side === "long") {
    return { paperAmount: size, creditAmount: -notional };
  }
  return { paperAmount: -size, creditAmount: notional };
}

export function buildMetaNodeOrder(input: BuildOrderInput): MetaNodeOrderPayload {
  const { paperAmount, creditAmount } = buildOrderAmounts(
    input.side,
    input.size,
    input.priceUsd
  );
  const expiration = input.expiration ?? Math.floor(Date.now() / 1000) + ORDER_TTL_SEC;
  const nonce =
    input.nonce ??
    Math.floor(Math.random() * 2 ** 32) + Math.floor(Date.now() / 1000);
  const info = packOrderInfo(
    MAKER_FEE_RATE,
    TAKER_FEE_RATE,
    BigInt(expiration),
    BigInt(nonce)
  );
  return {
    perp: input.perp,
    signer: input.signer,
    paperAmount,
    creditAmount,
    info,
    makerFeeRate: MAKER_FEE_RATE.toString(),
    takerFeeRate: TAKER_FEE_RATE.toString(),
    expiration,
    nonce,
  };
}

export function orderTypedData(
  order: MetaNodeOrderPayload,
  chainId: number = SEPOLIA_CHAIN_ID,
  dealer: Address = resolveDealerAddress()
) {
  return {
    domain: {
      name: "MetaNode",
      version: "1",
      chainId,
      verifyingContract: dealer,
    },
    types: METANODE_ORDER_TYPES,
    primaryType: "Order" as const,
    message: {
      perp: order.perp,
      signer: order.signer,
      paperAmount: order.paperAmount,
      creditAmount: order.creditAmount,
      info: order.info,
    },
  };
}

export function toCreateOrderBody(
  order: MetaNodeOrderPayload,
  signature: Hex
) {
  return {
    perp: order.perp,
    signer: order.signer,
    paperAmount: order.paperAmount.toString(),
    creditAmount: order.creditAmount.toString(),
    makerFeeRate: order.makerFeeRate,
    takerFeeRate: order.takerFeeRate,
    expiration: order.expiration,
    nonce: order.nonce,
    signature,
  };
}

/** 根据持仓构造平仓单（反向 paper/credit） */
export function buildCloseOrderInput(
  perp: Address,
  signer: Address,
  paperRaw: string,
  priceUsd: string
): BuildOrderInput {
  const paper = BigInt(paperRaw);
  if (paper === 0n) {
    throw new Error("仓位为 0");
  }
  const size = (paper < 0n ? -paper : paper).toString();
  const sizeHuman = (Number(size) / 1e18).toString();
  return {
    perp,
    signer,
    side: paper > 0n ? "short" : "long",
    size: sizeHuman,
    priceUsd,
  };
}
