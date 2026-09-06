import type { Abi } from "viem";

/** Sepolia 测试 USDC（公开 mint 接口） */
export const SEPOLIA_TEST_USDC =
  "0xf3B23a25F2ef5cD41E35eC6B48F97397d0d85dc0" as const;

export const sepoliaTestUsdcAbi = [
  {
    type: "function",
    name: "mint",
    inputs: [
      { name: "to", type: "address" },
      { name: "amount", type: "uint256" },
    ],
    outputs: [],
    stateMutability: "nonpayable",
  },
  {
    type: "function",
    name: "balanceOf",
    inputs: [{ name: "account", type: "address" }],
    outputs: [{ name: "", type: "uint256" }],
    stateMutability: "view",
  },
  {
    type: "function",
    name: "decimals",
    inputs: [],
    outputs: [{ name: "", type: "uint8" }],
    stateMutability: "view",
  },
  {
    type: "function",
    name: "transfer",
    inputs: [
      { name: "to", type: "address" },
      { name: "amount", type: "uint256" },
    ],
    outputs: [{ name: "", type: "bool" }],
    stateMutability: "nonpayable",
  },
  {
    type: "function",
    name: "approve",
    inputs: [
      { name: "spender", type: "address" },
      { name: "amount", type: "uint256" },
    ],
    outputs: [{ name: "", type: "bool" }],
    stateMutability: "nonpayable",
  },
  {
    type: "function",
    name: "allowance",
    inputs: [
      { name: "owner", type: "address" },
      { name: "spender", type: "address" },
    ],
    outputs: [{ name: "", type: "uint256" }],
    stateMutability: "view",
  },
] as const satisfies Abi;

/** 与后端 Ethereum.UsdcAddress 一致；未配置 env 时用 Sepolia 测试 USDC */
export function resolveUsdcAddress(): `0x${string}` {
  const raw = process.env.NEXT_PUBLIC_USDC_ADDRESS?.trim();
  if (raw && /^0x[a-fA-F0-9]{40}$/.test(raw)) {
    return raw as `0x${string}`;
  }
  return SEPOLIA_TEST_USDC;
}

/** 与后端 Ethereum.UsdcTreasuryAddress 一致的平台托管收款地址 */
export function resolveTreasuryAddress(): `0x${string}` | null {
  const raw = process.env.NEXT_PUBLIC_USDC_TREASURY_ADDRESS?.trim();
  if (raw && /^0x[a-fA-F0-9]{40}$/.test(raw)) {
    return raw as `0x${string}`;
  }
  return null;
}
