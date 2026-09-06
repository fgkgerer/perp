# ============================================
# MetaNode 永续合约教学项目 - Makefile
# ============================================
# 
# 使用说明：
#   make help          - 显示所有可用命令
#   make build         - 编译合约
#   make test          - 运行测试
#   make deploy-dealer - 部署 Dealer 合约
#
# 环境变量（部署前需设置）：
#   export MetaNode_DEPLOYER_PK=<你的私钥>
#   export RPC_URL=<RPC节点地址>
# ============================================

# 默认 RPC URL（可通过环境变量覆盖）
RPC_URL ?= http://localhost:8545
# 链 ID（可选，用于验证）
CHAIN_ID ?= 1

# Forge 通用参数
FORGE_OPTS := --via-ir

.PHONY: all build test clean fmt help install snapshot gas

# ============================================
# 基础命令
# ============================================

## 默认目标：编译
all: build

## 安装依赖
install:
	@echo "📦 安装 Foundry 依赖..."
	forge install

## 编译合约
build:
	@echo "🔨 编译合约..."
	forge build

## 清理构建产物
clean:
	@echo "🧹 清理构建产物..."
	forge clean
	rm -rf out cache

## 格式化代码
fmt:
	@echo "✨ 格式化代码..."
	forge fmt

## 检查格式
fmt-check:
	@echo "🔍 检查代码格式..."
	forge fmt --check

# ============================================
# 测试命令
# ============================================

## 运行所有测试
test:
	@echo "🧪 运行测试..."
	forge test -vvv

## 运行测试（简洁输出）
test-short:
	@echo "🧪 运行测试..."
	forge test

## 运行特定测试文件
test-file:
	@echo "🧪 运行测试文件: $(FILE)"
	forge test --match-path $(FILE) -vvv

## 运行特定测试函数
test-func:
	@echo "🧪 运行测试函数: $(FUNC)"
	forge test --match-test $(FUNC) -vvv

## 运行测试并生成 gas 报告
test-gas:
	@echo "⛽ 运行测试并生成 gas 报告..."
	forge test --gas-report

## 生成测试覆盖率报告
coverage:
	@echo "📊 生成覆盖率报告..."
	forge coverage

## 生成 gas 快照
snapshot:
	@echo "📸 生成 gas 快照..."
	forge snapshot

# ============================================
# 部署命令 - 本地测试网
# ============================================

## 启动本地 Anvil 节点
anvil:
	@echo "🔧 启动本地 Anvil 节点..."
	anvil

## 部署测试 ERC20 代币（本地）
deploy-test-token-local:
	@echo "🪙 部署测试代币到本地..."
	forge script script/deployTestERC20Token.s.sol:TokenScript \
		--rpc-url http://localhost:8545 \
		--broadcast \
		--private-key 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80

## 部署 MetaNodeDealer（本地）
deploy-dealer-local:
	@echo "🏦 部署 MetaNodeDealer 到本地..."
	forge script script/deployMetaNodeDealer.s.sol:MetaNodeDealerScript \
		--rpc-url http://localhost:8545 \
		--broadcast \
		--private-key 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80

## 部署 SubaccountFactory（本地）
deploy-factory-local:
	@echo "🏭 部署 SubaccountFactory 到本地..."
	forge script script/deployFactory.s.sol:SubaccountFactoryScript \
		--rpc-url http://localhost:8545 \
		--broadcast \
		--private-key 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80

## 部署 Perpetual（本地）
deploy-perpetual-local:
	@echo "📈 部署 Perpetual 到本地..."
	forge script script/deployPerpetual.s.sol:PerpetualScript \
		--rpc-url http://localhost:8545 \
		--broadcast \
		--private-key 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80

## 部署 EmergencyOracle（本地）
deploy-emergency-oracle-local:
	@echo "🔮 部署 EmergencyOracle 到本地..."
	forge script script/deployEmergencyOracle.s.sol:EmergencyOracleScript \
		--rpc-url http://localhost:8545 \
		--broadcast \
		--private-key 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80

# ============================================
# 部署命令 - 测试网/主网
# 需要设置环境变量: MetaNode_DEPLOYER_PK, RPC_URL
# ============================================

## 部署测试 ERC20 代币
deploy-test-token:
	@echo "🪙 部署测试代币..."
	forge script script/deployTestERC20Token.s.sol:TokenScript \
		--rpc-url $(RPC_URL) \
		--broadcast \
		--verify

## 部署 MetaNodeDealer
deploy-dealer:
	@echo "🏦 部署 MetaNodeDealer..."
	forge script script/deployMetaNodeDealer.s.sol:MetaNodeDealerScript \
		--rpc-url $(RPC_URL) \
		--broadcast \
		--verify

## 部署 SubaccountFactory
deploy-factory:
	@echo "🏭 部署 SubaccountFactory..."
	forge script script/deployFactory.s.sol:SubaccountFactoryScript \
		--rpc-url $(RPC_URL) \
		--broadcast \
		--verify

## 部署 Perpetual
deploy-perpetual:
	@echo "📈 部署 Perpetual..."
	forge script script/deployPerpetual.s.sol:PerpetualScript \
		--rpc-url $(RPC_URL) \
		--broadcast \
		--verify

## 部署 OracleAdaptor
deploy-oracle:
	@echo "🔮 部署 OracleAdaptor..."
	forge script script/deployOracleAdaptor.s.sol:OracleAdaptorScript \
		--rpc-url $(RPC_URL) \
		--broadcast \
		--verify

## 部署 EmergencyOracle
deploy-emergency-oracle:
	@echo "🆘 部署 EmergencyOracle..."
	forge script script/deployEmergencyOracle.s.sol:EmergencyOracleScript \
		--rpc-url $(RPC_URL) \
		--broadcast \
		--verify

# ============================================
# 一键部署（本地测试网）
# ============================================

## 部署完整系统到本地测试网
deploy-all-local: deploy-test-token-local deploy-dealer-local deploy-factory-local deploy-emergency-oracle-local
	@echo "✅ 本地部署完成!"
	@echo ""
	@echo "下一步操作："
	@echo "1. 记录部署的合约地址"
	@echo "2. 调用 MetaNodeDealer.setInsurance() 设置保险账户"
	@echo "3. 调用 MetaNodeDealer.setFundingRateKeeper() 设置资金费率更新者"
	@echo "4. 调用 MetaNodeDealer.setOrderSender() 设置撮合引擎"
	@echo "5. 部署 Perpetual 合约并调用 setPerpRiskParams() 注册市场"

# ============================================
# 实用工具
# ============================================

## 查看合约大小
size:
	@echo "📏 合约大小..."
	forge build --sizes

## 查看存储布局
storage:
	@echo "💾 存储布局..."
	forge inspect MetaNodeDealer storage-layout

## 生成 ABI
abi:
	@echo "📄 生成 ABI..."
	forge inspect MetaNodeDealer abi > out/MetaNodeDealer.abi.json
	forge inspect Perpetual abi > out/Perpetual.abi.json
	@echo "ABI 已生成到 out/ 目录"

## 更新依赖
update:
	@echo "🔄 更新依赖..."
	forge update

# ============================================
# 帮助信息
# ============================================

## 显示帮助
help:
	@echo ""
	@echo "╔══════════════════════════════════════════════════════════════════╗"
	@echo "║        MetaNode 永续合约教学项目 - 可用命令                       ║"
	@echo "╠══════════════════════════════════════════════════════════════════╣"
	@echo "║  基础命令:                                                        ║"
	@echo "║    make build            编译合约                                ║"
	@echo "║    make test             运行测试 (详细输出)                      ║"
	@echo "║    make test-short       运行测试 (简洁输出)                      ║"
	@echo "║    make clean            清理构建产物                            ║"
	@echo "║    make fmt              格式化代码                              ║"
	@echo "║    make install          安装依赖                                ║"
	@echo "╠══════════════════════════════════════════════════════════════════╣"
	@echo "║  测试命令:                                                        ║"
	@echo "║    make test-gas         测试并生成 gas 报告                      ║"
	@echo "║    make coverage         生成覆盖率报告                          ║"
	@echo "║    make snapshot         生成 gas 快照                           ║"
	@echo "║    make test-file FILE=<path>   运行指定测试文件                 ║"
	@echo "║    make test-func FUNC=<name>   运行指定测试函数                 ║"
	@echo "╠══════════════════════════════════════════════════════════════════╣"
	@echo "║  本地部署 (Anvil):                                                ║"
	@echo "║    make anvil            启动本地 Anvil 节点                      ║"
	@echo "║    make deploy-all-local 一键部署到本地                          ║"
	@echo "║    make deploy-dealer-local     部署 MetaNodeDealer              ║"
	@echo "║    make deploy-factory-local    部署 SubaccountFactory           ║"
	@echo "║    make deploy-perpetual-local  部署 Perpetual                   ║"
	@echo "╠══════════════════════════════════════════════════════════════════╣"
	@echo "║  测试网/主网部署 (需设置环境变量):                                 ║"
	@echo "║    export MetaNode_DEPLOYER_PK=<私钥>                            ║"
	@echo "║    export RPC_URL=<RPC地址>                                      ║"
	@echo "║    make deploy-dealer    部署 MetaNodeDealer                     ║"
	@echo "║    make deploy-factory   部署 SubaccountFactory                  ║"
	@echo "║    make deploy-perpetual 部署 Perpetual                          ║"
	@echo "║    make deploy-oracle    部署 OracleAdaptor                      ║"
	@echo "╠══════════════════════════════════════════════════════════════════╣"
	@echo "║  工具命令:                                                        ║"
	@echo "║    make size             查看合约大小                            ║"
	@echo "║    make storage          查看存储布局                            ║"
	@echo "║    make abi              生成 ABI 文件                           ║"
	@echo "║    make update           更新依赖                                ║"
	@echo "╚══════════════════════════════════════════════════════════════════╝"
	@echo ""

