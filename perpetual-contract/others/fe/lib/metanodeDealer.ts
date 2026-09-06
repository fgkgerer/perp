import type { Abi } from "viem";

/** Sepolia MetaNodeDealer（与 README / 后端 metanode.yaml 一致） */
export const SEPOLIA_DEALER =
  "0x62e738C8e807c5D8224044207ff7623F9e080Cd7" as const;

export const metaNodeDealerAbi = [
  {
    type: "function",
    name: "deposit",
    inputs: [
      { name: "primaryAmount", type: "uint256" },
      { name: "secondaryAmount", type: "uint256" },
      { name: "to", type: "address" },
    ],
    outputs: [],
    stateMutability: "nonpayable",
  },
  {
    type: "function",
    name: "getCreditOf",
    inputs: [{ name: "trader", type: "address" }],
    outputs: [
      { name: "primaryCredit", type: "int256" },
      { name: "secondaryCredit", type: "uint256" },
      { name: "pendingPrimaryWithdraw", type: "uint256" },
      { name: "pendingSecondaryWithdraw", type: "uint256" },
      { name: "executionTimestamp", type: "uint256" },
    ],
    stateMutability: "view",
  },
] as const satisfies Abi;

export function resolveDealerAddress(): `0x${string}` {
  const raw = process.env.NEXT_PUBLIC_DEALER_ADDRESS?.trim();
  if (raw && /^0x[a-fA-F0-9]{40}$/.test(raw)) {
    return raw as `0x${string}`;
  }
  return SEPOLIA_DEALER;
}
