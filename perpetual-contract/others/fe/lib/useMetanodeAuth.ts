"use client";

import { useCallback, useSyncExternalStore } from "react";
import {
  getMetanodeAuthPhase,
  hasMetanodeAuthSession,
  subscribeMetanodeAuth,
  type MetanodeAuthPhase,
} from "@/lib/metanode-api";

export function useMetanodeAuth(address?: string): {
  sessionOk: boolean;
  authPhase: MetanodeAuthPhase;
  authPending: boolean;
} {
  const subscribe = useCallback((onChange: () => void) => subscribeMetanodeAuth(onChange), []);

  const getSessionOk = useCallback(() => {
    if (!address) return false;
    return hasMetanodeAuthSession(address);
  }, [address]);

  const sessionOk = useSyncExternalStore(subscribe, getSessionOk, () => false);

  const getPhase = useCallback(() => getMetanodeAuthPhase(), []);
  const authPhase = useSyncExternalStore(subscribe, getPhase, () => "idle" as MetanodeAuthPhase);

  return {
    sessionOk,
    authPhase,
    authPending: authPhase === "pending",
  };
}
