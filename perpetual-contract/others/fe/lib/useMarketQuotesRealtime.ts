"use client";

import { useEffect, useRef, useState } from "react";
import { getSupabaseClient, isSupabaseConfigured } from "@/lib/supabase";
import { rowToQuote, type MarketQuoteRow } from "@/lib/market-quote";
import type { RealtimeChannel } from "@supabase/supabase-js";

export type RealtimeStatus =
  | "idle"
  | "connecting"
  | "subscribed"
  | "error"
  | "disabled";

type Options = {
  onQuote: (quote: MarketQuoteRow) => void;
  enabled?: boolean;
};

/** 订阅 Supabase Realtime：public.market_quotes 变更 */
export function useMarketQuotesRealtime({ onQuote, enabled = true }: Options) {
  const handlerRef = useRef(onQuote);
  handlerRef.current = onQuote;

  const [status, setStatus] = useState<RealtimeStatus>(
    enabled && isSupabaseConfigured() ? "connecting" : "disabled"
  );
  const [lastError, setLastError] = useState<string | null>(null);

  useEffect(() => {
    if (!enabled || !isSupabaseConfigured()) {
      setStatus("disabled");
      return;
    }

    const supabase = getSupabaseClient();
    if (!supabase) {
      setStatus("error");
      setLastError("Supabase client 初始化失败");
      return;
    }

    let cancelled = false;
    let channel: RealtimeChannel | null = null;

    setStatus("connecting");
    setLastError(null);

    void (async () => {
      const { data, error } = await supabase.from("market_quotes").select("*");
      if (cancelled) return;
      if (error) {
        setLastError(error.message);
        setStatus("error");
        return;
      }
      for (const row of data ?? []) {
        const q = rowToQuote(row as Record<string, unknown>);
        if (q) handlerRef.current(q);
      }
    })();

    channel = supabase
      .channel("metanode-market-quotes")
      .on(
        "postgres_changes",
        { event: "INSERT", schema: "public", table: "market_quotes" },
        (payload) => {
          const q = rowToQuote((payload.new ?? {}) as Record<string, unknown>);
          if (q) handlerRef.current(q);
        }
      )
      .on(
        "postgres_changes",
        { event: "UPDATE", schema: "public", table: "market_quotes" },
        (payload) => {
          const q = rowToQuote((payload.new ?? {}) as Record<string, unknown>);
          if (q) handlerRef.current(q);
        }
      )
      .subscribe((subscribeStatus, err) => {
        if (cancelled) return;
        if (subscribeStatus === "SUBSCRIBED") {
          setStatus("subscribed");
          setLastError(null);
          return;
        }
        if (
          subscribeStatus === "CHANNEL_ERROR" ||
          subscribeStatus === "TIMED_OUT"
        ) {
          setStatus("error");
          setLastError(err?.message ?? subscribeStatus);
        }
      });

    return () => {
      cancelled = true;
      if (channel) void supabase.removeChannel(channel);
    };
  }, [enabled]);

  return { status, lastError };
}
