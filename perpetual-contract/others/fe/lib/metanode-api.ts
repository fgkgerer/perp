/** MetaNode 后端 HTTP（钱包登录等） */

/** 生产默认 API（Vercel 未注入 env 时仍走公网后端，避免构建产物请求 localhost） */
const PROD_METANODE_API = "https://perpetual-api.zood.work";
const DEV_METANODE_API = "http://127.0.0.1:28888";

function resolveMetanodeApiBase(): string {
  const fromEnv = process.env.NEXT_PUBLIC_METANODE_API_URL?.trim();
  if (fromEnv) return fromEnv.replace(/\/$/, "");
  return process.env.NODE_ENV === "production"
    ? PROD_METANODE_API
    : DEV_METANODE_API;
}

export const METANODE_API_BASE = resolveMetanodeApiBase();

export const METANODE_TOKEN_KEY = "metanode_token";
/** JWT exp（Unix 秒），与 verify 返回的 expiresAt 一致时可由后端写入 */
export const METANODE_TOKEN_EXP_KEY = "metanode_token_exp";
/** 上次登录的钱包地址（小写），用于换账号时强制重登 */
export const METANODE_AUTH_WALLET_KEY = "metanode_auth_wallet";

/** 默认提前多少秒视为过期并刷新（避免卡点请求失败） */
export const METANODE_TOKEN_LEEWAY_SEC = 120;

export function getMetanodeToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem(METANODE_TOKEN_KEY);
}

function base64UrlDecodeJson(segment: string): Record<string, unknown> | null {
  try {
    const pad = "===".slice((segment.length + 3) % 4);
    const b64 = segment.replace(/-/g, "+").replace(/_/g, "/") + pad;
    const json = atob(b64);
    return JSON.parse(json) as Record<string, unknown>;
  } catch {
    return null;
  }
}

/** 从 JWT payload 解析 exp（秒）；非 JWT 返回 null */
export function decodeJwtExp(token: string): number | null {
  const parts = token.split(".");
  if (parts.length < 2) return null;
  const payload = base64UrlDecodeJson(parts[1]);
  if (!payload || typeof payload.exp !== "number") return null;
  return payload.exp;
}

/** 当前 token 过期时间（Unix 秒）；优先本地缓存，否则解析 JWT */
export function getMetanodeTokenExpiry(token?: string | null): number | null {
  if (typeof window === "undefined") return null;
  const raw = token ?? getMetanodeToken();
  if (!raw) return null;
  const cached = localStorage.getItem(METANODE_TOKEN_EXP_KEY);
  if (cached) {
    const n = Number.parseInt(cached, 10);
    if (Number.isFinite(n)) return n;
  }
  return decodeJwtExp(raw);
}

/**
 * 会话是否在有效期内（含提前量）。
 * 无法解析 exp 时视为仍有效（兼容非标准 token），依赖换钱包或手动清缓存刷新。
 */
export function isMetanodeSessionValid(
  leewaySec: number = METANODE_TOKEN_LEEWAY_SEC
): boolean {
  const tok = getMetanodeToken();
  if (!tok) return false;
  const exp = getMetanodeTokenExpiry(tok);
  if (exp == null) return true;
  const now = Math.floor(Date.now() / 1000);
  return now + leewaySec < exp;
}

/** 本地记录的登录钱包是否与当前连接地址一致 */
export function sessionMatchesWallet(currentAddress: string): boolean {
  if (typeof window === "undefined") return false;
  const w = localStorage.getItem(METANODE_AUTH_WALLET_KEY);
  if (!w) return true;
  return w.toLowerCase() === currentAddress.trim().toLowerCase();
}

/** 是否已有本地 token 且与当前钱包一致（不因 JWT 过期主动重登，由 401 触发） */
export function hasMetanodeAuthSession(currentAddress: string): boolean {
  return Boolean(getMetanodeToken()) && sessionMatchesWallet(currentAddress);
}

export const METANODE_AUTH_CHANGE = "metanode-auth-change";

export type MetanodeAuthPhase = "idle" | "pending" | "ready";

let metanodeAuthPhase: MetanodeAuthPhase = "idle";

export function getMetanodeAuthPhase(): MetanodeAuthPhase {
  return metanodeAuthPhase;
}

export function setMetanodeAuthPhase(phase: MetanodeAuthPhase): void {
  if (metanodeAuthPhase === phase) return;
  metanodeAuthPhase = phase;
  notifyMetanodeAuthChange();
}

const authChangeListeners = new Set<() => void>();

/** localStorage token / 登录阶段变化时通知 React 订阅方 */
export function notifyMetanodeAuthChange(): void {
  authChangeListeners.forEach((fn) => fn());
  if (typeof window !== "undefined") {
    window.dispatchEvent(new Event(METANODE_AUTH_CHANGE));
  }
}

export function subscribeMetanodeAuth(onChange: () => void): () => void {
  authChangeListeners.add(onChange);
  if (typeof window !== "undefined") {
    window.addEventListener(METANODE_AUTH_CHANGE, onChange);
  }
  return () => {
    authChangeListeners.delete(onChange);
    if (typeof window !== "undefined") {
      window.removeEventListener(METANODE_AUTH_CHANGE, onChange);
    }
  };
}

export function setMetanodeSession(
  token: string,
  expiresAtUnix?: number,
  walletLower?: string
): void {
  localStorage.setItem(METANODE_TOKEN_KEY, token);

  let exp = expiresAtUnix;
  if (typeof exp !== "number" || !Number.isFinite(exp)) {
    exp = decodeJwtExp(token) ?? undefined;
  }
  if (typeof exp === "number" && Number.isFinite(exp)) {
    localStorage.setItem(METANODE_TOKEN_EXP_KEY, String(Math.floor(exp)));
  } else {
    localStorage.removeItem(METANODE_TOKEN_EXP_KEY);
  }

  if (walletLower) {
    localStorage.setItem(METANODE_AUTH_WALLET_KEY, walletLower.trim().toLowerCase());
  }
  setMetanodeAuthPhase("ready");
}

export function clearMetanodeToken(): void {
  if (typeof window === "undefined") return;
  localStorage.removeItem(METANODE_TOKEN_KEY);
  localStorage.removeItem(METANODE_TOKEN_EXP_KEY);
  localStorage.removeItem(METANODE_AUTH_WALLET_KEY);
  setMetanodeAuthPhase("idle");
  notifyMetanodeAuthChange();
}

export type AuthNonceResp = {
  code: number;
  message: string;
  nonce?: string;
  messageToSign?: string;
};

export type AuthVerifyResp = {
  code: number;
  message: string;
  token?: string;
  expiresAt?: number;
  userId?: number;
  wallet?: string;
};

export async function postAuthNonce(address: string): Promise<AuthNonceResp> {
  const res = await fetch(`${METANODE_API_BASE}/api/v1/auth/nonce`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ address }),
    cache: "no-store",
  });
  const data = (await res.json()) as AuthNonceResp;
  if (!res.ok && data.code === undefined) {
    throw new Error(`nonce HTTP ${res.status}`);
  }
  return data;
}

export async function postAuthVerify(
  address: string,
  message: string,
  signature: `0x${string}`
): Promise<AuthVerifyResp> {
  const res = await fetch(`${METANODE_API_BASE}/api/v1/auth/verify`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ address, message, signature }),
    cache: "no-store",
  });
  const data = (await res.json()) as AuthVerifyResp;
  if (!res.ok && data.code === undefined) {
    throw new Error(`verify HTTP ${res.status}`);
  }
  return data;
}

type UnauthorizedHandler = () => void;
let unauthorizedHandler: UnauthorizedHandler | null = null;

/** WalletAuthSync 注册：API 返回 401 时触发重新 nonce → 签名 → verify */
export function setMetanodeUnauthorizedHandler(
  handler: UnauthorizedHandler | null
): void {
  unauthorizedHandler = handler;
}

function notifyUnauthorized(): void {
  unauthorizedHandler?.();
}

function isUnauthorizedResponse(res: Response, body?: { code?: number }): boolean {
  if (res.status === 401) return true;
  return body?.code === 401;
}

/** 带 Bearer 的请求；HTTP 401 或 body.code===401 时清 token 并通知重登 */
async function metanodeAuthRequest<T extends { code?: number }>(
  path: string,
  init?: RequestInit
): Promise<T> {
  const headers = new Headers(init?.headers);
  const t = getMetanodeToken();
  if (t) {
    headers.set("Authorization", `Bearer ${t}`);
  }
  const res = await fetch(`${METANODE_API_BASE}${path}`, {
    ...init,
    headers,
    cache: "no-store",
  });
  const text = await res.text();
  let data: T;
  try {
    data = (text ? JSON.parse(text) : {}) as T;
  } catch {
    if (res.status === 503 && text.includes("Request Timeout")) {
      throw new Error(
        "后端处理超时（Supabase 跨境延迟），请稍后刷新订单列表；若重复提交提示已存在则说明已成功"
      );
    }
    throw new Error(`HTTP ${res.status}: ${text.slice(0, 200)}`);
  }
  if (isUnauthorizedResponse(res, data)) {
    clearMetanodeToken();
    notifyUnauthorized();
  }
  return data;
}

export type AccountBalanceDTO = {
  trader: string;
  primaryCredit: string;
  onChainPrimaryCredit?: string;
  ledgerPrimaryBalance?: string;
  secondaryCredit: string;
  pendingPrimaryWithdraw: string;
  pendingSecondaryWithdraw: string;
  executionTimestamp: number;
};

export type GetBalanceRespDTO = {
  code: number;
  message: string;
  balance: AccountBalanceDTO;
};

export type DepositRecordDTO = {
  txHash: string;
  trader: string;
  primaryAmount: string;
  secondaryAmount: string;
  blockNumber: number;
  createTime: number;
};

export type ListDepositsRespDTO = {
  code: number;
  message: string;
  deposits: DepositRecordDTO[];
  total: number;
  page: number;
  pageSize: number;
};

export async function fetchMetanodeBalance(
  trader: string
): Promise<GetBalanceRespDTO> {
  const q = new URLSearchParams({ trader });
  return metanodeAuthRequest<GetBalanceRespDTO>(`/api/v1/balance?${q}`);
}

export async function fetchMetanodeDeposits(
  trader: string,
  page = 1,
  pageSize = 20
): Promise<ListDepositsRespDTO> {
  const q = new URLSearchParams({
    trader,
    page: String(page),
    pageSize: String(pageSize),
  });
  return metanodeAuthRequest<ListDepositsRespDTO>(`/api/v1/deposits?${q}`);
}

export type PerpMarketDTO = {
  address: string;
  name: string;
  markPrice: string;
  indexPrice: string;
  fundingRate: string;
  isRegistered: boolean;
  /** 链上 initialMarginRatio（1e18 原始值） */
  initialMarginRatio?: string;
  /** 初始保证金率 %，如 5.00 */
  initialMarginPct?: string;
  /** 最大杠杆倍数，如 20 */
  maxLeverage?: string;
  liquidationThreshold?: string;
  /** 维持保证金率 %，如 3.00 */
  maintenanceMarginPct?: string;
  liquidationPriceOff?: string;
  insuranceFeeRate?: string;
};

export type ListMarketsRespDTO = {
  code: number;
  message: string;
  markets: PerpMarketDTO[];
};

export type PositionDTO = {
  trader: string;
  perp: string;
  perpName: string;
  paper: string;
  credit: string;
  entryPrice: string;
  markPrice: string;
  liqPrice: string;
  updateTime: number;
};

export type GetPositionsRespDTO = {
  code: number;
  message: string;
  positions: PositionDTO[];
};

export type RiskInfoDTO = {
  trader: string;
  netValue: string;
  exposure: string;
  maintenanceMargin: string;
  availableMargin: string;
  marginRatio: string;
  isSafe: boolean;
};

export type GetRiskRespDTO = {
  code: number;
  message: string;
  riskInfo: RiskInfoDTO;
};

export type CreateOrderBody = {
  perp: string;
  signer: string;
  paperAmount: string;
  creditAmount: string;
  makerFeeRate: string;
  takerFeeRate: string;
  expiration: number;
  nonce: number;
  signature: string;
};

export type CreateOrderRespDTO = {
  code: number;
  message: string;
  orderId?: string;
};

export type OrderDTO = {
  orderId: string;
  perp: string;
  signer: string;
  paperAmount: string;
  creditAmount: string;
  status: number;
  filledAmount: string;
  createTime: number;
};

export type ListOrdersRespDTO = {
  code: number;
  message: string;
  orders: OrderDTO[];
  total: number;
};

export type KlineDTO = {
  time: number;
  open: string;
  high: string;
  low: string;
  close: string;
  volume: string;
};

export type GetKlinesRespDTO = {
  code: number;
  message: string;
  klines: KlineDTO[];
};

/** 前端图表周期 → 后端 interval（15/60/240/1D → 15m/1h/4h/1d） */
export function resolutionToKlineInterval(resolution: string): string {
  const r = resolution.trim();
  if (r === "60") return "1h";
  if (r === "240") return "4h";
  if (r.toUpperCase() === "1D") return "1d";
  if (r === "15") return "15m";
  return r;
}

export async function fetchMetanodeKlines(
  perp: string,
  resolution: string,
  opts?: { limit?: number; startTime?: number; endTime?: number }
): Promise<GetKlinesRespDTO> {
  const q = new URLSearchParams({
    perp,
    interval: resolutionToKlineInterval(resolution),
    limit: String(opts?.limit ?? 200),
  });
  if (opts?.startTime != null) {
    q.set("startTime", String(opts.startTime));
  }
  if (opts?.endTime != null) {
    q.set("endTime", String(opts.endTime));
  }
  const res = await fetch(`${METANODE_API_BASE}/api/v1/klines?${q}`, {
    cache: "no-store",
  });
  return (await res.json()) as GetKlinesRespDTO;
}

export async function fetchMetanodeMarkets(): Promise<ListMarketsRespDTO> {
  const res = await fetch(`${METANODE_API_BASE}/api/v1/markets`, {
    cache: "no-store",
  });
  return (await res.json()) as ListMarketsRespDTO;
}

export type OpenQuoteDTO = {
  perp: string;
  side: string;
  priceUsd: string;
  priceRaw: string;
  source: string;
  bestBid: string;
  bestAsk: string;
  markPrice: string;
  indexPrice: string;
};

export type GetOpenQuoteRespDTO = {
  code: number;
  message: string;
  quote: OpenQuoteDTO;
};

/** 开仓系统价：long=卖一，short=买一 */
export async function fetchMetanodeOpenQuote(
  perp: string,
  side: "long" | "short"
): Promise<GetOpenQuoteRespDTO> {
  const q = new URLSearchParams({ perp, side });
  const res = await fetch(`${METANODE_API_BASE}/api/v1/open-quote?${q}`, {
    cache: "no-store",
  });
  return (await res.json()) as GetOpenQuoteRespDTO;
}

export type OrderPreviewFillDTO = {
  priceUsd: string;
  priceRaw: string;
  amount: string;
  amountRaw: string;
};

export type OrderPreviewDTO = {
  perp: string;
  side: string;
  slippageBps: number;
  referencePriceUsd: string;
  referencePriceRaw: string;
  referenceSource: string;
  limitPriceUsd: string;
  limitPriceRaw: string;
  requestedSize: string;
  filledSize: string;
  unfilledSize: string;
  avgFillPriceUsd: string;
  worstFillPriceUsd: string;
  fullyFillable: boolean;
  fills: OrderPreviewFillDTO[];
};

export type GetOrderPreviewRespDTO = {
  code: number;
  message: string;
  preview: OrderPreviewDTO;
};

/** 下单前滑点/深度模拟（签名限价 = 参考价 ± slippageBps） */
export async function fetchMetanodeOrderPreview(
  perp: string,
  side: "long" | "short",
  size: string,
  slippageBps: number,
  signer?: string
): Promise<GetOrderPreviewRespDTO> {
  const q = new URLSearchParams({
    perp,
    side,
    size,
    slippageBps: String(slippageBps),
  });
  if (signer) q.set("signer", signer);
  const res = await fetch(`${METANODE_API_BASE}/api/v1/order-preview?${q}`, {
    cache: "no-store",
  });
  return (await res.json()) as GetOrderPreviewRespDTO;
}

export async function fetchMetanodePositions(
  trader: string
): Promise<GetPositionsRespDTO> {
  const q = new URLSearchParams({ trader });
  return metanodeAuthRequest<GetPositionsRespDTO>(`/api/v1/positions?${q}`);
}

export async function fetchMetanodeRisk(
  trader: string
): Promise<GetRiskRespDTO> {
  const q = new URLSearchParams({ trader });
  return metanodeAuthRequest<GetRiskRespDTO>(`/api/v1/risk?${q}`);
}

export async function postMetanodeOrder(
  body: CreateOrderBody
): Promise<CreateOrderRespDTO> {
  return metanodeAuthRequest<CreateOrderRespDTO>(`/api/v1/orders`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
}

export async function fetchMetanodeOrders(
  signer: string,
  perp?: string
): Promise<ListOrdersRespDTO> {
  const q = new URLSearchParams({ signer, page: "1", pageSize: "20" });
  if (perp) q.set("perp", perp);
  return metanodeAuthRequest<ListOrdersRespDTO>(`/api/v1/orders?${q}`);
}

export type OrderBookEntryDTO = {
  price: string;
  amount: string;
};

export type GetOrderBookRespDTO = {
  code: number;
  message: string;
  bids: OrderBookEntryDTO[];
  asks: OrderBookEntryDTO[];
};

export type TradeRecordDTO = {
  tradeId: string;
  perp: string;
  takerOrderId: string;
  makerOrderId: string;
  taker: string;
  maker: string;
  paperAmount: string;
  price: string;
  takerFee: string;
  makerFee: string;
  txHash: string;
  blockNumber: number;
  createTime: number;
};

export type ListTradesRespDTO = {
  code: number;
  message: string;
  trades: TradeRecordDTO[];
  total: number;
  page: number;
  pageSize: number;
};

export async function fetchMetanodeOrderBook(
  perp: string,
  limit = 20
): Promise<GetOrderBookRespDTO> {
  const q = new URLSearchParams({ perp, limit: String(limit) });
  const res = await fetch(`${METANODE_API_BASE}/api/v1/orderbook?${q}`, {
    cache: "no-store",
  });
  return (await res.json()) as GetOrderBookRespDTO;
}

export async function fetchMetanodeTrades(
  perp: string,
  pageSize = 50
): Promise<ListTradesRespDTO> {
  const q = new URLSearchParams({
    perp,
    page: "1",
    pageSize: String(pageSize),
  });
  const res = await fetch(`${METANODE_API_BASE}/api/v1/trades?${q}`, {
    cache: "no-store",
  });
  return (await res.json()) as ListTradesRespDTO;
}
