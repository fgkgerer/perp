// SPDX-License-Identifier: BUSL-1.1
pragma solidity ^0.8.19;

import "forge-std/Script.sol";
import "forge-std/console2.sol";

import "../src/MetaNodeDealer.sol";
import "../src/Perpetual.sol";
import "../src/libraries/Types.sol";
import "../src/support/TestERC20.sol";
import "../src/support/TestMarkPriceSource.sol";

/**
 * @title Sepolia 一键部署脚本
 * @notice 部署测试 USDC（可选）、MetaNodeDealer、2 个 Perp 市场、TestMarkPriceSource，并完成 Dealer 基础配置。
 *
 * ## 环境变量
 * - MetaNode_DEPLOYER_PK（必填）：部署者私钥，64 位十六进制字符串，**可带或不带 `0x` 前缀**
 * - MetaNode_ORDER_SENDER（可选）：撮合/上链 `Perpetual.trade` 的 EOA，须能调用链上成交；默认 = 部署者
 * - MetaNode_FUNDING_KEEPER（可选）：可调用 `updateFundingRate` 的地址；默认 = 部署者
 * - MetaNode_INSURANCE（可选）：保险账户；默认 = 部署者
 * - MetaNode_USDC_ADDRESS（可选）：若已存在 Sepolia 上的 USDC，填入则跳过部署 TestERC20，直接用该地址作为 `primaryAsset`
 *
 * ## 命令示例
 * ```bash
 * cd /path/to/smart-contract-EVM
 * export FOUNDRY_PROFILE=sepolia   # 必须：开启 via_ir，否则 MetaNodeDealer 超过 24KB 无法在链上部署
 * export MetaNode_DEPLOYER_PK=<你的私钥>
 * export SEPOLIA_RPC_URL=https://sepolia.infura.io/v3/<KEY>
 * forge script script/DeploySepolia.s.sol:DeploySepoliaScript \
 *   --rpc-url $SEPOLIA_RPC_URL \
 *   --broadcast \
 *   -vvvv
 * ```
 *
 * 模拟（不发链）：去掉 `--broadcast`。
 *
 * ## 合约说明
 * - TestMarkPriceSource：任意人可 `setMarkPrice`，仅适合测试网；生产请换预言机。
 * - 标记价精度与测试一致：使用 1e6 量级（与 USDC 6 位小数）写入 source，与 `TradingInit` 一致。
 *
 * 部署完成后，将日志中的 Dealer / Perp 地址填入后端 `others/backend/etc/metanode.yaml` 的 `DealerAddress` 与 `Markets.Address`。
 */
contract DeploySepoliaScript is Script {
    function test() public { }

    /// @dev 私钥：支持 `0x` 前缀或 64 位 hex 无前缀（与常见 .env / cast 写法一致）
    function _envPrivateKey(string memory name) internal view returns (uint256) {
        string memory raw = vm.envString(name);
        bytes memory b = bytes(raw);
        if (b.length >= 2 && b[0] == bytes1("0") && (b[1] == bytes1("x") || b[1] == bytes1("X"))) {
            return vm.parseUint(raw);
        }
        return vm.parseUint(string.concat("0x", raw));
    }

    function run() external {
        uint256 pk = _envPrivateKey("MetaNode_DEPLOYER_PK");
        address deployer = vm.addr(pk);

        address orderSender = vm.envOr("MetaNode_ORDER_SENDER", deployer);
        address fundingKeeper = vm.envOr("MetaNode_FUNDING_KEEPER", deployer);
        address insurance = vm.envOr("MetaNode_INSURANCE", deployer);

        vm.startBroadcast(pk);

        address usdc;
        try vm.envString("MetaNode_USDC_ADDRESS") returns (string memory usdcEnv) {
            if (bytes(usdcEnv).length > 0) {
                usdc = vm.parseAddress(usdcEnv);
            } else {
                usdc = address(new TestERC20("USDC", "USDC", 6));
            }
        } catch {
            usdc = address(new TestERC20("USDC", "USDC", 6));
        }

        MetaNodeDealer dealer = new MetaNodeDealer(usdc);

        dealer.setMaxPositionAmount(20);
        dealer.setOrderSender(orderSender, true);
        dealer.setInsurance(insurance);
        dealer.setFundingRateKeeper(fundingKeeper);

        TestMarkPriceSource pxBtc = new TestMarkPriceSource();
        TestMarkPriceSource pxEth = new TestMarkPriceSource();

        Perpetual perpBtc = new Perpetual(address(dealer));
        Perpetual perpEth = new Perpetual(address(dealer));

        Types.RiskParams memory pBtc = Types.RiskParams({
            initialMarginRatio: 5e16,
            liquidationThreshold: 3e16,
            liquidationPriceOff: 1e16,
            insuranceFeeRate: 1e16,
            markPriceSource: address(pxBtc),
            name: "BTC-PERP",
            isRegistered: true
        });
        Types.RiskParams memory pEth = Types.RiskParams({
            initialMarginRatio: 1e17,
            liquidationThreshold: 5e16,
            liquidationPriceOff: 1e16,
            insuranceFeeRate: 2e16,
            markPriceSource: address(pxEth),
            name: "ETH-PERP",
            isRegistered: true
        });

        dealer.setPerpRiskParams(address(perpBtc), pBtc);
        dealer.setPerpRiskParams(address(perpEth), pEth);

        pxBtc.setMarkPrice(30_000e6);
        pxEth.setMarkPrice(2000e6);

        vm.stopBroadcast();

        console2.log("========== MetaNode Sepolia ==========");
        console2.log("ChainId", block.chainid);
        console2.log("Deployer", deployer);
        console2.log("OrderSender (set validOrderSender)", orderSender);
        console2.log("FundingKeeper", fundingKeeper);
        console2.log("Insurance", insurance);
        console2.log("USDC (primaryAsset)", usdc);
        console2.log("MetaNodeDealer", address(dealer));
        console2.log("Perp BTC-PERP", address(perpBtc));
        console2.log("Perp ETH-PERP", address(perpEth));
        console2.log("MarkPrice BTC source", address(pxBtc));
        console2.log("MarkPrice ETH source", address(pxEth));
    }
}
