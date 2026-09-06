# MetaNode 后端服务

基于 go-zero 框架的去中心化永续合约交易系统后端。

## 项目结构

```
backend/
├── metanode.api          # API 定义文件
├── metanode.go           # 主入口文件
├── go.mod                # Go 模块文件
├── etc/
│   └── metanode.yaml     # 配置文件
├── internal/
│   ├── config/           # 配置结构
│   ├── handler/          # HTTP 处理器 (自动生成)
│   ├── logic/            # 业务逻辑
│   │   ├── order/        # 订单逻辑
│   │   ├── trade/        # 交易逻辑
│   │   ├── position/     # 仓位逻辑
│   │   ├── account/      # 账户逻辑
│   │   ├── market/       # 市场逻辑
│   │   ├── funding/      # 资金费率逻辑
│   │   └── liquidation/  # 清算逻辑
│   ├── model/            # 数据模型
│   ├── svc/              # 服务上下文
│   ├── types/            # 类型定义 (自动生成)
│   └── engine/           # 核心引擎
│       ├── matchengine.go      # 撮合引擎
│       ├── liquidator.go       # 清算机器人
│       └── fundingratekeeper.go # 资金费率服务
└── doc/
    └── sql/
        └── schema.sql    # 数据库表结构
```

## 快速开始

### 1. 安装依赖

```bash
# 安装 goctl 工具
go install github.com/zeromicro/go-zero/tools/goctl@latest

# 安装项目依赖
go mod tidy
```

### 2. 生成代码

```bash
# 根据 API 定义生成代码
goctl api go -api metanode.api -dir .
```

### 3. 初始化数据库

```bash
# 创建数据库并导入表结构
mysql -u root -p < doc/sql/schema.sql
```

### 4. 配置环境

编辑 `etc/metanode.yaml`，配置：
- MySQL 连接信息
- Redis 连接信息
- 以太坊 RPC 节点
- 合约地址
- 私钥（用于撮合引擎提交交易）

### 5. 启动服务

```bash
go run metanode.go -f etc/metanode.yaml
```

## API 接口

### 订单相关

| 方法 | 路径 | 说明 |
|-----|------|------|
| POST | /api/v1/orders | 创建订单 |
| GET | /api/v1/orders/:orderId | 查询订单 |
| GET | /api/v1/orders | 查询订单列表 |
| POST | /api/v1/orders/cancel | 取消订单 |

### 交易相关

| 方法 | 路径 | 说明 |
|-----|------|------|
| GET | /api/v1/trades | 查询成交记录 |

### 仓位相关

| 方法 | 路径 | 说明 |
|-----|------|------|
| GET | /api/v1/positions | 查询用户仓位 |
| GET | /api/v1/risk | 查询风险信息 |

### 账户相关

| 方法 | 路径 | 说明 |
|-----|------|------|
| GET | /api/v1/balance | 查询账户余额 |
| GET | /api/v1/deposits | 查询存款记录 |
| GET | /api/v1/withdraws | 查询取款记录 |

### 市场相关

| 方法 | 路径 | 说明 |
|-----|------|------|
| GET | /api/v1/markets | 获取所有市场 |
| GET | /api/v1/markets/:address | 获取市场详情 |
| GET | /api/v1/klines | 获取K线数据 |
| GET | /api/v1/ticker | 获取行情 |
| GET | /api/v1/orderbook | 获取深度 |

### 资金费率

| 方法 | 路径 | 说明 |
|-----|------|------|
| GET | /api/v1/funding-rates | 获取资金费率历史 |

### 清算

| 方法 | 路径 | 说明 |
|-----|------|------|
| GET | /api/v1/liquidations | 获取清算记录 |

## 核心组件

### 1. 撮合引擎 (MatchEngine)

负责订单撮合和链上交易提交：

```go
// 启动撮合引擎
matchEngine := engine.NewMatchEngine(cfg, db)
matchEngine.Start()

// 添加订单
matchEngine.AddOrder(order)
```

功能：
- 维护订单簿（买单/卖单分开排序）
- 价格-时间优先撮合
- 批量提交链上交易
- 更新订单状态

### 2. 清算机器人 (Liquidator)

监控不安全仓位并执行清算：

```go
// 启动清算机器人
liquidator := engine.NewLiquidator(cfg, ethCfg, db)
liquidator.Start()
```

功能：
- 定期检查所有仓位的安全状态
- 自动执行清算交易
- 记录清算记录

### 3. 资金费率服务 (FundingRateKeeper)

定期结算资金费率：

```go
// 启动资金费率服务
fundingKeeper := engine.NewFundingRateKeeper(cfg, ethCfg, markets, db)
fundingKeeper.Start()
```

功能：
- 获取标记价格和指数价格
- 计算资金费率
- 提交到链上
- 记录历史

## 订单签名

前端需要使用 EIP-712 签名订单：

```javascript
const domain = {
  name: "MetaNode",
  version: "1",
  chainId: chainId,
  verifyingContract: dealerAddress
};

const types = {
  Order: [
    { name: "perp", type: "address" },
    { name: "signer", type: "address" },
    { name: "paperAmount", type: "int128" },
    { name: "creditAmount", type: "int128" },
    { name: "info", type: "bytes32" }
  ]
};

// info = makerFeeRate(8) + takerFeeRate(8) + expiration(8) + nonce(8)
const info = ethers.utils.solidityPack(
  ["int64", "int64", "uint64", "uint64"],
  [makerFeeRate, takerFeeRate, expiration, nonce]
);

const order = {
  perp: perpAddress,
  signer: userAddress,
  paperAmount: "1000000000000000000",  // 1 BTC
  creditAmount: "-30000000000",         // -$30,000
  info: info
};

const signature = await signer._signTypedData(domain, types, order);
```

## 开发说明

### 添加新接口

1. 在 `metanode.api` 中定义类型和路由
2. 运行 `goctl api go -api metanode.api -dir .`
3. 在 `internal/logic/` 中实现业务逻辑

### 数据库迁移

修改 `doc/sql/schema.sql` 后，手动执行 SQL 或使用迁移工具。

### 测试

```bash
go test ./...
```

## License

BUSL-1.1

