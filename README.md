# MetaNode

去中心化永续合约交易系统（教学项目）。核心模型：**链下撮合、链上结算** —— 用户用 EIP-712 签名订单，relayer 在内存撮合，只有撮合成功的 `trade()` 才上链。

## 组件

| 组件 | 路径 | 技术栈 | 说明 |
|---|---|---|---|
| 智能合约 | [`perpetual-contract/`](perpetual-contract/) | Solidity 0.8.19 / Foundry | `Perpetual.sol` + `MetaNodeDealer.sol`（见 [README](perpetual-contract/README.md)） |
| 后端 | [`perp-backend/`](perp-backend/) 与 [`perpetual-contract/others/backend/`](perpetual-contract/others/backend/) | Go 1.24 / go-zero + GORM | 撮合引擎 + 资金费 + 清算 + 充值监听（见 [README](perp-backend/README.md)） |
| 前端 | [`perpetual-contract/others/fe/`](perpetual-contract/others/fe/) | Next.js + wagmi + supabase | 交易 / 持仓 / 水龙头 |

> ⚠️ 后端存在两份拷贝：`perp-backend/` 是快照，`perpetual-contract/others/backend/` 更完整（含 funding/market/orderbook/fillsim 等）。改后端前先确认权威版本。

## 文档导航

| 文档 | 说明 |
|---|---|
| [`docs/architecture.md`](docs/architecture.md) | **整体架构与流程说明**：架构图（ASCII + Mermaid）、角色权限、前端↔后端↔合约三条链路、用户操作流程、四大链上流程 |
| [`CLAUDE_ZH.md`](CLAUDE_ZH.md) | Claude Code 项目说明（中文）：命令、架构、注意事项 |
| [`CLAUDE.md`](CLAUDE.md) | 同上（英文版） |
| [`perpetual-contract/docs/`](perpetual-contract/docs/) | 合约设计文档、时序图、业务逻辑解析 |
| [`perpetual-contract/others/backend/docs/`](perpetual-contract/others/backend/docs/) | 撮合引擎 / 订单簿结构 / 限价·市价单 / 滑点保护方案（中文） |
| [`perpetual-contract/others/docs/`](perpetual-contract/others/docs/) | 仓位存储与交易、8 小时资金费实现方案（中文） |

## 快速开始

各组件独立开发，构建/测试命令见各自 README：

- **合约**：`cd perpetual-contract && make build && make test`
- **后端**：`cd perp-backend && make run`（配置见 `etc/metanode.yaml`）
- **前端**：`cd perpetual-contract/others/fe && npm run dev`

> 更完整的命令清单（单测、部署、代码生成等）见 [`CLAUDE_ZH.md`](CLAUDE_ZH.md) 或各子项目 README。

## License

BUSL-1.1
