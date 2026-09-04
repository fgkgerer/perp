# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this repository is

"MetaNode" — a decentralized perpetual-contract trading system (teaching/educational project). The core model is **off-chain order matching, on-chain settlement**: users sign orders with EIP-712, a relayer matches them, and only the matched `trade()` is submitted on-chain. There are three components:

- `perpetual-contract/` — Solidity smart contracts (Foundry).
- `perp-backend/` — Go backend (go-zero REST + GORM).
- `perpetual-contract/others/fe/` — Next.js frontend.

### ⚠️ Two copies of the backend exist

The backend is checked in twice:

- `perp-backend/` — a partial snapshot.
- `perpetual-contract/others/backend/` — a **more complete** copy (adds `internal/funding/`, `internal/market/`, `internal/engine/orderbook.go` + `fillsim.go` + test suites, and `internal/logic/market/` files like `markprice.go`, `slippage.go`, `orderpreviewlogic.go`, `riskparams.go`).

When changing backend code, confirm which copy is authoritative before editing, and don't fix a bug in only one copy. The frontend (`perpetual-contract/others/fe/`) lives under the same `others/` tree.

## Smart contracts (`perpetual-contract/`)

Foundry project. `foundry.toml` uses `solc_version = 0.8.19`, optimizer `200` runs. Remappings (via `lib/`): `forge-std`, `@chainlink`, `@openzeppelin`.

```bash
make build          # forge build
make test           # forge test -vvv
make test-short     # forge test (quiet)
make test-file FILE=test/impl/DealerDecimal6Test.sol   # single file
make test-func FUNC=testBalanceCheck                  # single test
make test-gas       # --gas-report
make coverage       # forge coverage
make fmt            # forge fmt
make anvil          # local node (accounts logged; RPC http://localhost:8545)
make deploy-all-local   # deploy token + dealer + factory + emergency oracle locally
make deploy-dealer      # needs MetaNode_DEPLOYER_PK + RPC_URL env vars
```

**Contract size:** the `MetaNodeDealer` is near Ethereum's 24 KB limit. Production/Sepolia builds must use the `sepolia` profile (`FOUNDRY_PROFILE=sepolia forge build` / `make` scripts use `--via-ir`). The default profile does **not** enable `via_ir`.

CI: `.github/workflows/test.yml` runs `forge build --sizes` and `forge test -vvv` on push (submodules recursive).

### Architecture — two core contracts, Dealer split across files

- **`src/Perpetual.sol`** — the balance sheet for one market. A trader's position is `paper` (asset amount) + `credit`; both may be negative (e.g. long 1 BTC @ $30k → `paper=1, credit=-30000`; short → `paper=-1, credit=30000`). Actual credit = `paper * fundingRate + reducedCredit`.
- **`src/MetaNodeDealer.sol`** — owns multiple `Perpetual` markets. Only three operations change a balance: **funding rate, trading, liquidation**. The Dealer is split into five files that share one contract:
  - `MetaNodeDealer.sol` (entry), `MetaNodeStorage.sol` (state vars), `MetaNodeExternal.sol` (external fns — incl. `approveTrade`), `MetaNodeOperation.sol` (owner-only), `MetaNodeView.sol` (view fns).
- **Libraries** (`src/libraries/`) carry the real math: `Trading.sol`, `Liquidation.sol`, `Funding.sol`, `Position.sol`, `Operation.sol`, `Types.sol`, `Errors.sol`, `EIP712.sol`, `SignedDecimalMath.sol`.
- **Oracles** (`src/oracle/`): `ConstOracle`, `EmergencyOracle`, `OracleAdaptor`, `PythOracleAdaptor`.
- **Subaccounts** (`src/subaccount/`): `Subaccount.sol` + `SubaccountFactory.sol` (`newSubaccount()`); a subaccount is a contract whose owner is a user wallet, used to segregate risk/positions.
- **EIP-712 order**: domain `{name:"MetaNode", version:"1", chainId, verifyingContract: dealer}`; order `{perp, signer, paperAmount(int128), creditAmount(int128), info(bytes32)}`. `info` packs `makerFeeRate + takerFeeRate + expiration + nonce` (8 bytes each). See `perp-backend/internal/chain/eip712.go` and `internal/logic/auth/message.go` for the Go encoding; tests use `test/utils/EIP712Test.sol`.

Test layout: `test/init/TradingInit.sol` (base fixture: deploys USDC/MUSD, Dealer, ConstOracle, BTC-PERP + ETH-PERP, sets order-sender), `test/impl/` (behavioral + `Scenario*` suites), `test/mocks/`, `test/utils/`.

## Backend (`perp-backend/` and `perpetual-contract/others/backend/`)

Go 1.24, module `metanode`, built on **go-zero** REST. Entry point `metanode.go`; run with `go run metanode.go -f etc/metanode.yaml` (or `make run`).

```bash
make gen      # goctl api go -api metanode.api -dir .  (regenerate handlers/types from the .api spec)
make build    # go build -o bin/metanode metanode.go
make run      # go run metanode.go -f etc/metanode.yaml
make test     # go test -v ./...   (single test: go test ./internal/engine -run TestX)
make lint     # golangci-lint run
make fmt      # go fmt ./...
```

`metanode.api` is the goctl API spec — the single source of truth for request/response types and routes. **Never hand-edit generated files** (`internal/handler/*.go` handlers, `internal/types/types.go`, `internal/handler/routes.go` — all marked "Code generated by goctl"); change `metanode.api` and re-run `make gen`. Business logic lives in `internal/logic/<group>/<handler>logic.go`; handlers call into it.

### Backend architecture

`metanode.go` wires everything into a `rest.Server`, then starts long-running background engines:

- `engine.MatchEngine` — in-memory order book, price-time matching, submits matched trades on-chain. Exposed via `ServiceContext` as `MatchEngine` (order intake, `AddOrder`) and `OrderBook` (depth snapshots).
- `engine.Liquidator` — periodically checks positions, liquidates unsafe ones.
- `engine.FundingRateKeeper` — settles funding rate every 8h (`FundingRate.SettleInterval`).
- `listener.CoinbaseFeed` — Coinbase WebSocket ticker → Supabase `market_quotes` (drives frontend realtime + index price; also `ServiceContext.IndexPrice`).
- `listener.TreasuryDepositWatcher` — scans USDC `Transfer` to `Ethereum.UsdcTreasuryAddress`, credits `deposits`/`ledger_balances` (Redis stores scan cursor).

**The chain client is optional.** `internal/chain.NewClient` failure sets `ctx.Chain = nil`; the HTTP API still serves (on-chain queries return empty/zero) but matching and funding settlement stop. Config in `etc/metanode.yaml` (`Ethereum.RpcUrl`, `DealerAddress`, `PrivateKey` must match the on-chain `validOrderSender`).

**Two DB layers coexist** (see `internal/svc/servicecontext.go`). Models split because go-zero `sqlx` uses `?` placeholders which are incompatible with pgx/Postgres:
- `go-zero sqlx` (`sqlx.SqlConn`) — `position`, `withdraw`, `funding_rate`, `liquidation`, `market`, `kline`.
- `GORM` over Supabase/Postgres — `deposit`, `ledger_balance`, `order`, `trade`, `market_quote` (`model.NewGorm*Model`).

DB config is `Supabase.DataSource` in YAML (Postgres URI, `sslmode=require`). Tables are auto-migrated at startup via `internal/db/migrate.go`; `doc/sql/schema.sql` and `schema/` are reference DDL (the README's `mysql -u root` instructions are stale — the live DB is Postgres, not MySQL).

Auth: wallet EIP-191 signature login (`/api/v1/auth/nonce`, `/api/v1/auth/verify`) → JWT (HS256), configured under `Auth:` in YAML. Routes beyond the generated set are registered in `internal/handler/wallet_auth_handlers.go`.

## Frontend (`perpetual-contract/others/fe/`)

Next.js (canary) + TypeScript + Tailwind. Wallet via `wagmi`/`viem`/`rainbowkit`; realtime via `@supabase/supabase-js`; charts via `lightweight-charts`; server state via `@tanstack/react-query`.

```bash
npm run dev     # next dev
npm run build   # next build
npm run lint    # next lint
```

Pages: `/` (trade UI), `/positions`, `/faucet`. Components under `components/` map to backend endpoints (`OrderBook`, `Positions`, `FundingRatePanel`, etc.). Backend base URL and Supabase keys come from env (see `others/fe/.env.example`).

## Cross-cutting notes

- Sepolia deployments are documented in `perpetual-contract/README.md` (Dealer, BTC-PERP, ETH-PERP, USDC addresses); the backend's `etc/metanode.yaml` `Markets:` must point at these addresses.
- Prices are represented with fixed-point decimals on-chain (`SignedDecimalMath.sol`) and as decimal strings over the API (`"paperAmount"`, `"creditAmount"` etc. are `string`, not integers).
- Deeper design notes live in `perpetual-contract/docs/` and `perpetual-contract/others/backend/docs/` (matching-engine, order-book structure, funding-rate implementation — in Chinese).
