# MetaNode: 去中心化永续合约教学项目

MetaNode 是一个基于链下撮合、链上结算的去中心化永续合约交易系统教学项目。

## 什么是永续合约？

永续合约是一种金融衍生品，参与者买卖虚拟票据，通常锚定外部参考价格（如"比特币价格"）。MetaNode 使用资金费率机制使合约价格与现货价格保持一致，为交易者提供比现货市场更高的流动性和杠杆。

## 核心架构

项目只有两个核心智能合约：[Perpetual.sol](./src/Perpetual.sol) 和 [MetaNodeDealer.sol](./src/MetaNodeDealer.sol)。

- `Perpetual.sol` 是某个永续合约市场的核心资产负债表
- `MetaNodeDealer.sol` 拥有 `Perpetual.sol`。一个 `MetaNodeDealer.sol` 可以同时拥有多个 `Perpetual.sol`

## Perpetual.sol: 资产负债表

对于交易者，其余额由 paper（资产数量）和 credit 组成。这两个值的组合构成其余额。paper 和 credit 都可以为负数。

示例：

- 以 $30,000 做多 1BTC:

```javascript
paper = 1;
credit = -30000;
```

- 以 $30,000 做空 1BTC:

```javascript
paper = -1;
credit = 30000;
```

永续合约计算的本质是余额的状态转移。只有三种类型的操作会影响余额：

1. 资金费率 (Funding Rate)
2. 交易 (Trading)
3. 清算 (Liquidation)

### 资金费率

资金费率确保永续合约价格与现货价格保持一致：

- 当合约价格高于现货价格时，多头被惩罚，空头获得奖励
- 当合约价格低于现货价格时，空头被惩罚，多头获得奖励

为了基于 paper 管理 credit 调整，记录一个名为 "reducedCredit" 的值。实际 credit 使用以下公式计算：

```
credit = (paper * fundingRate) + reducedCredit
```

查询余额的函数：

```solidity
function balanceOf(address trader)
    external
    view
    returns (int256 paper, int256 credit);
```

### 交易

Relayer 作为订单发送者，收集 maker 和 taker 的订单，匹配它们，然后使用 `trade` 函数提交到永续合约。只有经过验证的地址才能成为订单发送者。

验证和计算过程在 [MetaNodeExternal.sol](./src/MetaNodeExternal.sol) 的 `approveTrade` 中处理。

### 清算

清算是一种强制交易行为。你可以通过调用以下函数触发清算：

```solidity
function liquidate(
    address liquidator,
    address liquidatedTrader,
    int256 requestPaper,
    int256 expectCredit
) external returns (int256 liqtorPaperChange, int256 liqtorCreditChange)
```

## MetaNodeDealer.sol: 交易台

负责维护资金费率、执行交易和管理清算。

### 特性

- **链下撮合，链上结算**：用户生成签名订单并传输到服务器，匹配后提交到区块链
- **全仓模式**：不同市场的仓位共享保证金
- **固定折扣清算**：对于保证金率低的账户，以固定折扣出售其仓位

### 存款保证金

MetaNodeDealer 接受 USDC 作为保证金。存款和取款不需要服务器许可。

### 取款保证金

两种取款模式：
- **待定取款**：用户需要等待时间锁才能获得保证金
- **快速取款**：用户可以一步取出保证金

## 子账户系统

使用专门设计的合约作为交易账户，用户的钱包地址是该合约的所有者。子账户可以帮助其所有者管理风险和仓位。

参见 [SubaccountFactory.sol](./src/subaccount/SubaccountFactory.sol) 中的 `newSubaccount()`。

## 项目结构

```
src/
├── MetaNodeDealer.sol      # 核心交易系统入口
├── MetaNodeStorage.sol     # 存储变量
├── MetaNodeExternal.sol    # 外部调用函数
├── MetaNodeOperation.sol   # Owner 专用函数
├── MetaNodeView.sol        # 视图函数
├── Perpetual.sol           # 永续合约市场
├── interfaces/             # 接口定义
│   ├── IDealer.sol
│   ├── IPerpetual.sol
│   └── internal/
├── libraries/              # 核心库
│   ├── Trading.sol         # 交易逻辑
│   ├── Liquidation.sol     # 清算逻辑
│   ├── Funding.sol         # 资金相关
│   ├── Position.sol        # 仓位管理
│   ├── Operation.sol       # 操作函数
│   ├── Types.sol           # 类型定义
│   ├── Errors.sol          # 错误消息
│   ├── EIP712.sol          # 签名相关
│   └── SignedDecimalMath.sol
├── oracle/                 # 价格预言机
│   ├── OracleAdaptor.sol
│   ├── ConstOracle.sol
│   ├── EmergencyOracle.sol
│   └── PythOracleAdaptor.sol
└── subaccount/             # 子账户系统
    ├── Subaccount.sol
    └── SubaccountFactory.sol
```

---

## 快速开始

### 环境要求

- [Foundry](https://book.getfoundry.sh/getting-started/installation) - Solidity 开发框架
- Git

### 安装依赖

```shell
# 克隆项目
git clone <repo-url>
cd smart-contract-EVM

# 安装 Foundry 依赖
make install
# 或
forge install
```

### 编译合约

```shell
make build
# 或
forge build
```

### 运行测试

```shell
# 运行所有测试（详细输出）
make test

# 运行所有测试（简洁输出）
make test-short

# 运行指定测试文件
make test-file FILE=test/impl/DealerDecimal6Test.sol

# 运行指定测试函数
make test-func FUNC=testBalanceCheck

# 生成 gas 报告
make test-gas

# 生成覆盖率报告
make coverage
```

---

## 本地部署指南

### 步骤 1：启动本地测试网

在终端 1 中启动 Anvil 本地节点：

```shell
make anvil
# 或
anvil
```

Anvil 会输出 10 个测试账户，默认 RPC 地址为 `http://localhost:8545`。

### 步骤 2：一键部署（推荐）

在终端 2 中执行：

```shell
make deploy-all-local
```

这会依次部署：
1. 测试 ERC20 代币（USDC）
2. MetaNodeDealer
3. SubaccountFactory
4. EmergencyOracle

### 步骤 3：手动部署（分步）

如果需要分步部署，按以下顺序执行：

```shell
# 1. 部署测试代币
make deploy-test-token-local

# 2. 部署核心 Dealer 合约
make deploy-dealer-local

# 3. 部署子账户工厂
make deploy-factory-local

# 4. 部署紧急预言机
make deploy-emergency-oracle-local

# 5. 部署永续合约市场（需要先配置 Dealer 地址）
make deploy-perpetual-local
```

### 步骤 4：配置合约

部署完成后，需要进行以下配置：

```solidity
// 1. 设置保险账户
MetaNodeDealer.setInsurance(insuranceAddress);

// 2. 设置资金费率更新者
MetaNodeDealer.setFundingRateKeeper(keeperAddress, true);

// 3. 设置订单发送者（撮合引擎）
MetaNodeDealer.setOrderSender(relayerAddress, true);

// 4. 注册永续合约市场
MetaNodeDealer.setPerpRiskParams(perpAddress, riskParams);

// 5. 设置快速取款白名单（可选）
MetaNodeDealer.setFastWithdrawalWhitelist(address, true);
```

---

## 测试网/主网部署

### 环境变量配置

```shell
# 设置部署者私钥
export MetaNode_DEPLOYER_PK=<你的私钥>

# 设置 RPC URL
export RPC_URL=https://sepolia.infura.io/v3/<your-key>  # Sepolia 测试网
# 或
export RPC_URL=https://mainnet.infura.io/v3/<your-key>  # 以太坊主网
```

### 部署命令

```shell
# 部署 MetaNodeDealer
make deploy-dealer

# 部署 SubaccountFactory
make deploy-factory

# 部署 Perpetual
make deploy-perpetual

# 部署 OracleAdaptor
make deploy-oracle
```

---

## 测试用例说明

### 测试文件结构

```
test/
├── init/
│   └── TradingInit.sol      # 测试初始化基类
├── impl/
│   ├── DealerDecimal6Test.sol       # 6位小数精度测试
│   ├── DealerInternalLibraryTest.sol # 内部数学库测试
│   ├── DealerOperationTest.sol      # 管理操作测试
│   └── DealerViewTest.sol           # 视图函数测试
├── mocks/
│   ├── MockChainLink.t.sol   # Chainlink 预言机模拟
│   ├── MockERC1271.sol       # ERC-1271 签名验证模拟
│   ├── MockUSDCPrice.sol     # USDC 价格模拟
│   └── WETH9.sol             # WETH 模拟
└── utils/
    ├── Checkers.sol          # 测试辅助检查
    ├── Utils.sol             # 工具函数
    └── EIP712Test.sol        # EIP-712 签名测试
```

### 测试初始化流程（TradingInit.sol）

```
1. 部署 USDC 代币（主资产，6位小数）
2. 部署 MUSD 代币（次级资产，6位小数）
3. 部署 MetaNodeDealer(USDC)
4. 设置 MUSD 为次级资产
5. 部署价格源（ConstOracle）
6. 部署 BTC-PERP 和 ETH-PERP 市场
7. 配置风险参数并注册市场
8. 设置订单发送者权限
9. 为测试账户分配代币并授权
```

### 各测试文件功能

| 测试文件 | 测试内容 |
|---------|---------|
| `DealerDecimal6Test.sol` | 存款余额、交易后净值/敞口计算、清算价格计算 |
| `DealerInternalLibraryTest.sol` | 有符号小数乘除法、绝对值、EIP-712 域分隔符 |
| `DealerOperationTest.sol` | 市场注销、权限控制、风险参数验证、次级资产限制 |
| `DealerViewTest.sol` | 订单验证、取款验证、授权查询、版本查询 |

---

## Makefile 命令速查

运行 `make help` 查看所有可用命令：

| 分类 | 命令 | 说明 |
|-----|------|------|
| **基础** | `make build` | 编译合约 |
| | `make test` | 运行测试（详细） |
| | `make clean` | 清理构建 |
| | `make fmt` | 格式化代码 |
| **测试** | `make test-gas` | 生成 gas 报告 |
| | `make coverage` | 覆盖率报告 |
| | `make snapshot` | gas 快照 |
| **本地部署** | `make anvil` | 启动本地节点 |
| | `make deploy-all-local` | 一键部署 |
| | `make deploy-dealer-local` | 部署 Dealer |
| **工具** | `make size` | 合约大小 |
| | `make storage` | 存储布局 |
| | `make abi` | 生成 ABI |

---

## 流程时序图

详细的系统交互时序图请参见：[docs/sequence-diagram.drawio](./docs/sequence-diagram.drawio)

### 核心流程概览

```
┌─────────────────────────────────────────────────────────────────┐
│                    MetaNode 交易流程                             │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  用户A                   Relayer                  链上合约        │
│    │                       │                        │           │
│    │ 1. 签名订单(做多)      │                        │           │
│    │ ─────────────────────>│                        │           │
│    │                       │                        │           │
│  用户B                     │                        │           │
│    │ 2. 签名订单(做空)      │                        │           │
│    │ ─────────────────────>│                        │           │
│    │                       │                        │           │
│    │                       │ 3. 匹配订单             │           │
│    │                       │ ──────────────────────>│           │
│    │                       │                        │           │
│    │                       │    trade()             │           │
│    │                       │    Perpetual           │           │
│    │                       │                        │           │
│    │<─────────────────────────────────────仓位更新──│           │
│    │                       │                        │           │
└─────────────────────────────────────────────────────────────────┘
```

---

## License

BUSL-1.1


## sepolia 部署的合约地址
Sepolia 上相关合约已部署（链上 bytecode 已核对），本次用到地址如下：

角色	地址
TestERC20 USDC
0xf3B23a25F2ef5cD41E35eC6B48F97397d0d85dc0
MetaNodeDealer
0x62e738C8e807c5D8224044207ff7623F9e080Cd7
Perp BTC-PERP
0x11Aae1f92Ff10bfbb205971e060CF6d9D917723b
Perp ETH-PERP
0x98456DCbcEfea550293727A7E2DfD45De92740c0
MarkPrice BTC
0xa7b19B51498290225f01eCB4176967a2d7a7e647
MarkPrice ETH
0x7d10A364358982BC238ec4208264e846c1454096
浏览器示例：
[USDC](https://sepolia.etherscan.io/address/0xf3b23a25f2ef5cd41e35ec6b48f97397d0d85dc0)、
[Dealer](https://sepolia.etherscan.io/address/0x62e738c8e807c5d8224044207ff7623f9e080cd7)。

