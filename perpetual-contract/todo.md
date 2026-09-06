# MetaNode 永续合约跑通 Todo

> 目标：对照 `README.md`、`test/impl/ScenarioTradingTest.sol` 与 `others/backend`，跑通 Sepolia 最小交易闭环。

## Sepolia 关键地址

> 2026-05-27 重部署（含 `Trading.sol` expiration 解析修复），与 `others/backend/etc/metanode.yaml` / 根目录 `README.md` 一致。

| 角色 | 地址 |
|------|------|
| USDC | `0xf3B23a25F2ef5cD41E35eC6B48F97397d0d85dc0` |
| Dealer | `0x62e738C8e807c5D8224044207ff7623F9e080Cd7` |
| BTC-PERP | `0x11Aae1f92Ff10bfbb205971e060CF6d9D917723b` |
| ETH-PERP | `0x98456DCbcEfea550293727A7E2DfD45De92740c0` |
| MarkPrice BTC | `0xa7b19B51498290225f01eCB4176967a2d7a7e647` |
| MarkPrice ETH | `0x7d10A364358982BC238ec4208264e846c1454096` |
| Treasury | `0x149F9754dB43Dda309a0B52a802D23F52e04F4F9` |
| validOrderSender / 部署者 | `0x1a0137b97542f03Dd1ee7F9A6859F17940E22D40` |

后端 `PrivateKey` 须对应链上 **`validOrderSender`**。重部署后须向**新 Dealer** 重新 `deposit` 保证金，旧 Dealer 余额不迁移。

---

## P0 — 不完成无法成交

- [x] **Supabase 建表**：`orders`、`trades`（及后续需要的 `funding_rates` 等）
- [x] **订单/成交改 GORM**：`OrderModel`、`TradeModel` 从 sqlx 迁到 Postgres GORM（避免 `?` 占位符报错）
- [x] **启动 migrate 补全**：`others/backend/internal/db/migrate.go` 自动创建 `orders` / `trades`
- [x] **校验 `PrivateKey`**：对应地址在 Sepolia Dealer 上为 `validOrderSender`（`others/backend/etc/metanode.yaml`）
- [x] **链上 deposit 入口**：前端或脚本调 `MetaNodeDealer.deposit()`（不能只用 Treasury 转账）
- [x] **确认 Markets 配置**：`metanode.yaml` 中 BTC/ETH Perp 地址与 README Sepolia 一致，重启后端验证

---

## P1 — 最小交易闭环（对齐 ScenarioTradingTest 场景 1）

- [x] **前端 EIP-712 下单**：`signTypedData` + `POST /api/v1/orders`
- [x] **订单参数对齐测试**：paper 1e18、credit 6 位 USDC、makerFee=1e14、takerFee=5e14
- [x] **双账户对手盘**：Alice 做多 + Bob 做空，两笔订单能撮合（UI 已支持；需两钱包手动各下一笔反向单）
- [x] **验证撮合上链**：Etherscan 上 Perp 有 `trade` tx，`GET /api/v1/positions` 有仓位（持仓/风险面板已接 API）
- [x] **验证余额/风险**：`GET /api/v1/balance`、`GET /api/v1/risk` 与链上一致（持仓区展示 risk；账户页 balance）
- [x] **平仓流程**：反向订单撮合后仓位归零、PnL 合理（持仓「平仓」按钮预填反向 EIP-712 单）

### P1 手动验证步骤

1. 钱包 A、B 各在 Dealer 存入保证金（账户页「存入保证金」）
2. 钱包 A：BTC-PERP 做多 1 @ 标记价 → EIP-712 提交
3. 钱包 B：同市场做空 1 @ 同价 → 提交
4. 等待撮合（约 1–2s），刷新持仓，应看到 Perp `trade` tx
5. 任一方点「平仓」，再让对手盘反向挂单完成平仓

---

## P2 — 账户与运维

- [x] **区分两种充值 UI**：Treasury（链下 ledger）vs Dealer.deposit（链上保证金），文案写清楚
- [x] **Treasury watcher 稳定**：RPC 可用、Supabase 连通、扫链入账正常
- [x] **撮合引擎重启恢复**：从 DB 加载 pending 单回内存订单簿（可选但建议）
- [x] **链上失败不回写 DB**：`trade()` revert 时不应把订单标为已成交

---

## P3 — API / 产品补齐

- [ ] **实现 `POST /api/v1/orders/cancel`**（取消订单）
- [ ] **实现 `GET /api/v1/orderbook`**（深度，读内存簿或 DB）
- [ ] **实现 `GET /api/v1/trades`**（成交列表）
- [ ] **实现 `GET /api/v1/funding-rates`**（读 DB / 链上）
- [ ] **下单鉴权**：`CreateOrder` 校验 JWT，`signer` 与登录钱包一致

---

## P4 — 进阶（Scenario 3/4 之后）

- [ ] **Liquidator 实装**：扫不安全账户 → 调 `liquidate` → 写 `liquidations`
- [ ] **FundingRateKeeper 验证**：`PrivateKey` 是否为 `fundingRateKeeper`；Sepolia 无 Chainlink 时的降级策略
- [ ] **Mark price 运维**：TestMarkPriceSource 改价脚本/后台（模拟 Scenario 涨跌）
- [ ] **更新 `others/backend/README.md`**：MySQL → Supabase、Treasury vs Dealer deposit、Sepolia 跑通步骤

---

## 建议执行顺序

```
P0（基础设施）→ P1（能开平仓）→ P2（账户体验）→ P3（API 完整）→ P4（清算/资金费）
```

---

## 相关文件

| 模块 | 路径 |
|------|------|
| 合约测试参考 | `test/impl/ScenarioTradingTest.sol` |
| 根 README | `README.md` |
| 前端账户/充值 | `others/fe/components/UserProfile.tsx`、`UsdcDepositForm.tsx` |
| 前端 API | `others/fe/lib/metanode-api.ts` |
| 后端入口 | `others/backend/metanode.go` |
| 后端配置 | `others/backend/etc/metanode.yaml` |
| DB migrate | `others/backend/internal/db/migrate.go` |
| GORM 模型 | `others/backend/internal/model/pgstore.go` |
| 撮合引擎 | `others/backend/internal/engine/matchengine.go` |
| 链上交互 | `others/backend/internal/chain/` |

---

## 当前已知 Gap

1. ~~**Treasury 充值 ≠ 链上 `Dealer.deposit`**~~（P0 已加 Dealer 充值入口）
2. ~~**`orders` / `trades` 仍用 sqlx + `?`**~~（P0 已迁 GORM）
3. ~~**前端 OrderForm 未接 MetaNode EIP-712 下单**~~（P1 已完成）
4. **Liquidator、cancel / orderbook / trades API** 多为 stub
5. **链上 trade revert 时 DB 仍可能标为已成交**（P2）
