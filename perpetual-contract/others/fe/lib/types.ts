export type FuturesRow = {
  symbol: string;
  index_price: number;
  mark_price: number;
  sum_unitary_funding: number;
  est_funding_rate: number;
  last_funding_rate: number;
  next_funding_time: number;
  open_interest: number;
  "24h_open": number;
  "24h_close": number;
  "24h_high": number;
  "24h_low": number;
  "24h_volume": number;
  "24h_amount": number;
};

export type FuturesResponse = {
  success: boolean;
  data: {
    rows: FuturesRow[];
  };
  timestamp: number;
};

export type KlineResponse = {
  s: string;
  o: number[];
  c: number[];
  h: number[];
  l: number[];
  v: number[];
  t: number[];
};

export type KlinePoint = {
  time: number;
  open: number;
  high: number;
  low: number;
  close: number;
  volume: number;
};
