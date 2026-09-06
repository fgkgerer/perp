import { FuturesResponse, KlinePoint, KlineResponse } from "@/lib/types";

export const FUTURES_URL = "https://api.orderly.org/v1/public/futures";

export const buildKlineUrl = (
  symbol: string,
  resolution: string,
  from: number,
  to: number
) =>
  `https://api.orderly.org/v1/tv/history?symbol=${symbol}&resolution=${resolution}&from=${from}&to=${to}`;

export const fetchFutures = async () => {
  const response = await fetch(FUTURES_URL, { cache: "no-store" });
  if (!response.ok) {
    throw new Error(`futures 请求失败: ${response.status}`);
  }
  const data = (await response.json()) as FuturesResponse;
  if (!data.success) {
    throw new Error("futures 返回失败");
  }
  return data.data.rows;
};

export const fetchKlines = async (url: string) => {
  const response = await fetch(url, { cache: "no-store" });
  if (!response.ok) {
    throw new Error(`kline 请求失败: ${response.status}`);
  }
  const data = (await response.json()) as KlineResponse;
  if (data.s !== "ok") {
    throw new Error("kline 返回失败");
  }
  const points: KlinePoint[] = data.t.map((time, index) => ({
    time,
    open: data.o[index],
    high: data.h[index],
    low: data.l[index],
    close: data.c[index],
    volume: data.v[index]
  }));
  return points;
};
