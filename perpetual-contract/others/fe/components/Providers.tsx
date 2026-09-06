"use client";

import * as React from "react";
import {
  RainbowKitProvider,
  getDefaultConfig,
  darkTheme,
} from "@rainbow-me/rainbowkit";
import {
  base,
  sepolia,
} from "wagmi/chains";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { WagmiProvider, http } from "wagmi";
import WalletAuthSync from "@/components/WalletAuthSync";

/** 与后端 metanode.yaml / cast 一致；勿用 Infura 演示节点（易限流导致 mint 不上链） */
const DEFAULT_SEPOLIA_RPC = "https://ethereum-sepolia.publicnode.com";

const sepoliaRpc =
  typeof process.env.NEXT_PUBLIC_SEPOLIA_RPC_URL === "string" &&
  process.env.NEXT_PUBLIC_SEPOLIA_RPC_URL.trim() !== ""
    ? process.env.NEXT_PUBLIC_SEPOLIA_RPC_URL.trim()
    : DEFAULT_SEPOLIA_RPC;

const config = getDefaultConfig({
  appName: "MetaNode Perpetual",
  projectId: "YOUR_PROJECT_ID", // TODO: Get a project ID from https://cloud.walletconnect.com
  chains: [sepolia, base],
  transports: {
    [sepolia.id]: http(sepoliaRpc),
    [base.id]: http(),
  },
  ssr: true,
});

const queryClient = new QueryClient();

export function Providers({ children }: { children: React.ReactNode }) {
  return (
    <WagmiProvider config={config}>
      <QueryClientProvider client={queryClient}>
        <RainbowKitProvider 
            theme={darkTheme({
                accentColor: '#16D8D4',
                accentColorForeground: 'black',
                borderRadius: 'small',
                overlayBlur: 'small',
            })}
        >
          <WalletAuthSync />
          {children}
        </RainbowKitProvider>
      </QueryClientProvider>
    </WagmiProvider>
  );
}
