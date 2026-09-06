"use client";

import { useEffect, useRef } from "react";
import { useAccount, useSignMessage } from "wagmi";
import {
  clearMetanodeToken,
  hasMetanodeAuthSession,
  METANODE_API_BASE,
  postAuthNonce,
  postAuthVerify,
  setMetanodeAuthPhase,
  setMetanodeSession,
  setMetanodeUnauthorizedHandler,
} from "@/lib/metanode-api";

/**
 * 连接钱包后：若 localStorage 已有有效 token 则复用，否则 nonce → 签名 → verify 并持久化。
 * 后端 API 返回 401 时清 token 并重新走登录；仅在用户主动断开连接后清除本地会话。
 */
export default function WalletAuthSync() {
  const { address, isConnected, status } = useAccount();
  const { signMessageAsync } = useSignMessage();
  const inFlight = useRef(false);
  const wasConnected = useRef(false);

  useEffect(() => {
    if (status === "connecting" || status === "reconnecting") {
      return;
    }

    if (!isConnected || !address) {
      if (wasConnected.current && status === "disconnected") {
        clearMetanodeToken();
        wasConnected.current = false;
      }
      return;
    }

    wasConnected.current = true;
    let cancelled = false;

    const needsLogin = () => !hasMetanodeAuthSession(address);

    const runLogin = async () => {
      if (cancelled || inFlight.current || !address) return;
      if (!needsLogin()) {
        setMetanodeAuthPhase("ready");
        return;
      }

      inFlight.current = true;
      setMetanodeAuthPhase("pending");
      try {
        const n = await postAuthNonce(address);
        if (cancelled) return;
        if (n.code !== 0 || !n.messageToSign) {
          console.warn(
            "[MetaNode auth] nonce failed:",
            n.message ?? "unknown",
            `(${METANODE_API_BASE})`
          );
          setMetanodeAuthPhase("idle");
          return;
        }

        const sig = await signMessageAsync({
          message: n.messageToSign,
        });
        if (cancelled) return;

        const v = await postAuthVerify(address, n.messageToSign, sig);
        if (cancelled) return;
        if (v.code !== 0 || !v.token) {
          console.warn("[MetaNode auth] verify failed:", v.message ?? "unknown");
          setMetanodeAuthPhase("idle");
          return;
        }

        setMetanodeSession(v.token, v.expiresAt, address);
      } catch (e) {
        if (!cancelled) {
          console.warn("[MetaNode auth]", e instanceof Error ? e.message : e);
          setMetanodeAuthPhase("idle");
        }
      } finally {
        inFlight.current = false;
      }
    };

    void runLogin();

    setMetanodeUnauthorizedHandler(() => {
      if (!cancelled) void runLogin();
    });

    return () => {
      cancelled = true;
      setMetanodeUnauthorizedHandler(null);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps -- signMessageAsync 引用不稳
  }, [isConnected, address, status]);

  return null;
}
