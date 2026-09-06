# MetaNode 永续合约 - 仓位存储与开仓逻辑详解

## 一、仓位存储方式

### 1.1 核心存储结构 (Perpetual.sol)

```solidity
// 每个交易者的仓位用一个简单的结构体存储
struct balance {
    int128 paper;          // 仓位数量：正数=多头，负数=空头
    int128 reducedCredit;  // 约减后的资金量（用于 gas 优化）
}

// 存储映射：交易者地址 => 余额
mapping(address => balance) balanceMap;

// 当前累计资金费率
int256 fundingRate;
```

### 1.2 多头 vs 空头的区分

**核心原则：用 `paper` 的正负来区分多空**

| 仓位类型 | paper | credit | 含义 |
|---------|-------|--------|------|
| **多头 (Long)** | **正数** (+1e18) | **负数** (-30000e6) | 持有 1 BTC，欠 $30,000 |
| **空头 (Short)** | **负数** (-1e18) | **正数** (+30000e6) | 欠 1 BTC，持有 $30,000 |

**示例：**
```
Alice 做多 1 BTC @ $30,000:
  paper = +1e18 (持有 1 BTC)
  credit = -30000e6 (欠 $30,000)

Bob 做空 1 BTC @ $30,000:
  paper = -1e18 (欠 1 BTC)
  credit = +30000e6 (持有 $30,000)
```

### 1.3 为什么用 `reducedCredit` 而不是直接存 `credit`？

这是一个 **gas 优化技巧**，与资金费率机制相关：

```solidity
// 实际 credit 的计算公式：
credit = (paper × fundingRate) + reducedCredit

// 反推：
reducedCredit = credit - (paper × fundingRate)
```

**优势：** 当资金费率更新时，所有持仓者的 `credit` 会自动变化，**无需单独修改每个用户的存储**！

```
资金费率更新示例：
假设 fundingRate 从 0 增加到 0.01%

多头 (paper = +1 BTC):
  credit 自动增加 = 1 BTC × 0.01% = 收到 $3

空头 (paper = -1 BTC):
  credit 自动减少 = -1 BTC × 0.01% = 支付 $3
```

### 1.4 数据存储位置总结

```
┌─────────────────────────────────────────────────────────────┐
│                    Perpetual 合约 (BTC-PERP)                │
├─────────────────────────────────────────────────────────────┤
│  balanceMap[Alice] = { paper: +1e18, reducedCredit: xxx }  │
│  balanceMap[Bob]   = { paper: -1e18, reducedCredit: yyy }  │
│  fundingRate = 1e14  (累计资金费率)                         │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│                    MetaNodeDealer 合约                       │
├─────────────────────────────────────────────────────────────┤
│  primaryCredit[Alice] = 10000e6   (USDC 保证金余额)         │
│  primaryCredit[Bob]   = 10000e6                             │
│  openPositions[Alice] = [BTC-PERP, ETH-PERP]  (持仓市场)    │
│  openPositions[Bob]   = [BTC-PERP]                          │
└─────────────────────────────────────────────────────────────┘
```

---

## 二、开仓逻辑详解

### 2.1 完整的开仓数据流

```
┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐
│  用户    │    │  前端    │    │  后端    │    │ Perpetual│    │  Dealer  │
│ (Alice)  │    │          │    │ 撮合引擎 │    │  合约    │    │  合约    │
└────┬─────┘    └────┬─────┘    └────┬─────┘    └────┬─────┘    └────┬─────┘
     │               │               │               │               │
     │ 1.下单请求    │               │               │               │
     │──────────────>│               │               │               │
     │               │               │               │               │
     │ 2.构建订单    │               │               │               │
     │<──────────────│               │               │               │
     │               │               │               │               │
     │ 3.签名(EIP712)│               │               │               │
     │──────────────>│               │               │               │
     │               │               │               │               │
     │               │ 4.发送签名订单│               │               │
     │               │──────────────>│               │               │
     │               │               │               │               │
     │               │               │ 5.匹配订单    │               │
     │               │               │──────────────>│               │
     │               │               │ (trade)       │               │
     │               │               │               │               │
     │               │               │               │ 6.验证&计算   │
     │               │               │               │──────────────>│
     │               │               │               │ (approveTrade)│
     │               │               │               │               │
     │               │               │               │ 7.返回变化量  │
     │               │               │               │<──────────────│
     │               │               │               │               │
     │               │               │               │ 8.更新仓位    │
     │               │               │               │ (_settle)     │
     │               │               │               │               │
     │               │               │               │ 9.通知开仓    │
     │               │               │               │──────────────>│
     │               │               │               │ (openPosition)│
     │               │               │               │               │
     │               │               │               │ 10.安全检查   │
     │               │               │               │<──────────────│
     │               │               │               │ (isAllSafe)   │
```

### 2.2 订单结构 (Types.Order)

```solidity
struct Order {
    address perp;           // 目标永续合约地址 (BTC-PERP)
    address signer;         // 订单签名者 (交易者)
    int128 paperAmount;     // 仓位数量：正=做多，负=做空
    int128 creditAmount;    // 资金：与 paper 异号
    bytes32 info;           // 打包的附加信息
}

// info 字段布局 (256 bits):
// ╔═══════════════╤═══════════════╤═══════════════╤═══════════════╗
// ║ makerFeeRate  │ takerFeeRate  │  expiration   │    nonce      ║
// ║   (64 bits)   │   (64 bits)   │   (64 bits)   │   (64 bits)   ║
// ╚═══════════════╧═══════════════╧═══════════════╧═══════════════╝
```

**订单示例：**
```
Alice 想做多 1 BTC @ $30,000:
{
  perp: "0x...BTC_PERP",
  signer: "0x...Alice",
  paperAmount: +1e18,          // 正数 = 做多
  creditAmount: -30000e6,      // 负数 = 支付
  info: 0x...                  // makerFee=0.01%, takerFee=0.05%, exp=xxx
}

Bob 想做空 1 BTC @ $30,000:
{
  perp: "0x...BTC_PERP",
  signer: "0x...Bob",
  paperAmount: -1e18,          // 负数 = 做空
  creditAmount: +30000e6,      // 正数 = 收取
  info: 0x...
}
```

### 2.3 撮合逻辑 (Trading._matchOrders)

```solidity
// 撮合结果结构
struct MatchResult {
    address[] traderList;       // [Alice, Bob] 参与者
    int256[] paperChangeList;   // [+1e18, -1e18] 仓位变化
    int256[] creditChangeList;  // [-30015e6, +29997e6] 资金变化(含手续费)
    int256 orderSenderFee;      // 手续费收入
}
```

**撮合流程：**
1. **价格匹配验证**：Taker 和 Maker 方向相反，Maker 价格不劣于 Taker 限价
2. **使用 Maker 价格成交**：保护 Maker
3. **计算手续费**：
   - Maker: 0.01% × $30,000 = $3
   - Taker: 0.05% × $30,000 = $15

### 2.4 仓位结算 (Perpetual._settle)

```solidity
function _settle(address trader, int256 paperChange, int256 creditChange) internal {
    bool isNewPosition = balanceMap[trader].paper == 0;  // 是否新开仓
    
    // 1. 计算新的 credit
    int256 newCredit = 旧credit + creditChange;
    
    // 2. 计算新的 paper
    int128 newPaper = 旧paper + paperChange;
    
    // 3. 反推 reducedCredit 存储
    int128 newReducedCredit = newCredit - (newPaper × fundingRate);
    
    // 4. 更新存储
    balanceMap[trader].paper = newPaper;
    balanceMap[trader].reducedCredit = newReducedCredit;
    
    // 5. 如果是新开仓，通知 Dealer
    if (isNewPosition) {
        IDealer(owner()).openPosition(trader);  // 记录到持仓列表
    }
    
    // 6. 如果完全平仓，实现盈亏
    if (newPaper == 0) {
        IDealer(owner()).realizePnl(trader, newReducedCredit);
        balanceMap[trader].reducedCredit = 0;
    }
}
```

### 2.5 安全检查 (Liquidation._isSafe)

开仓后必须检查账户是否安全：

```
安全条件：netValue >= exposure × initialMarginRatio

netValue = primaryCredit + secondaryCredit + 未实现盈亏
exposure = Σ |paper × markPrice|  (所有仓位的敞口)

示例：
Alice 用 $10,000 保证金做多 1 BTC @ $30,000
- netValue = $10,000 - $15(手续费) = $9,985
- exposure = 1 BTC × $30,000 = $30,000
- 要求：$9,985 >= $30,000 × 5% = $1,500 ✓ 安全
```

---

## 三、关键代码路径

### 3.1 开仓调用链

```
Perpetual.trade(tradeData)
    │
    ├──> Dealer.approveTrade(sender, tradeData)    // 验证订单
    │        │
    │        ├──> 验证签名 (ECDSA.tryRecover)
    │        ├──> 验证过期时间
    │        ├──> 验证价格有效性 (paper 和 credit 异号)
    │        ├──> 更新订单已成交量
    │        ├──> Trading._matchOrders()            // 撮合计算
    │        └──> 返回 (traderList, paperChangeList, creditChangeList)
    │
    ├──> _settle(trader, paperChange, creditChange)  // 结算每个交易者
    │        │
    │        ├──> 更新 balanceMap
    │        ├──> Dealer.openPosition()  (如果是新仓位)
    │        └──> Dealer.realizePnl()    (如果完全平仓)
    │
    └──> Dealer.isAllSafe(traderList)               // 检查所有人安全
```

### 3.2 持仓数据查询

```solidity
// 查询某用户在 BTC-PERP 的仓位
(int256 paper, int256 credit) = btcPerp.balanceOf(alice);
// paper > 0 → 多头
// paper < 0 → 空头
// paper = 0 → 无仓位

// 查询用户的保证金和风险状态
(int256 netValue, uint256 exposure, , ) = dealer.getTraderRisk(alice);

// 查询清算价格
uint256 liqPrice = dealer.getLiquidationPrice(alice, address(btcPerp));
```

---

## 四、总结：为什么这样设计？

### 4.1 使用 paper/credit 双字段

- **paper**：直接表示仓位方向和数量，正负区分多空
- **credit**：记录资金变化，用于计算盈亏
- **优势**：简单直观，一个数字同时表达方向和大小

### 4.2 使用 reducedCredit 存储

- **目的**：优化资金费率更新的 gas 消耗
- **原理**：`credit = paper × fundingRate + reducedCredit`
- **优势**：更新资金费率只需改 1 个变量，不需遍历所有用户

### 4.3 链下撮合 + 链上结算

- **链下**：后端撮合引擎匹配订单，快速高效
- **链上**：验证签名、检查保证金、原子结算
- **优势**：兼顾性能和安全性

### 4.4 EIP-712 签名

- **用途**：用户离线签名订单，无需每次交易都上链
- **安全**：包含合约地址、链 ID，防止重放攻击
- **体验**：钱包可以展示人类可读的订单信息
