// SPDX-License-Identifier: GPL-2.0-or-later

pragma solidity ^0.8.19;

import "forge-std/Script.sol";
import "../src/oracle/OracleAdaptor.sol";

/**
 * @title OracleAdaptorScript - 部署预言机适配器
 * @notice 用于部署 Chainlink 预言机适配器
 * 
 * 部署命令：
 * forge script script/deployOracleAdaptor.s.sol:OracleAdaptorScript --rpc-url <RPC_URL> --broadcast
 * 
 * 环境变量：
 * - MetaNode_DEPLOYER_PK: 部署者私钥
 * 
 * 构造函数参数说明：
 * - chainlink: 资产/USD Chainlink 价格源地址
 * - decimalsCorrection: Chainlink 价格小数位数（通常为 8 或 10）
 * - heartbeatInterval: 资产价格心跳间隔（秒）
 * - usdcHeartbeat: USDC 价格心跳间隔（秒）
 * - usdcSource: USDC/USD Chainlink 价格源地址
 * - priceThreshold: 价格偏离阈值（1e18 基数）
 */
contract OracleAdaptorScript is Script {
    // 排除在覆盖率报告之外
    function test() public { }

    /**
     * @notice 执行部署
     * @dev 部署时需要根据目标链配置正确的参数：
     *      - Chainlink 价格源地址因链而异
     *      - 心跳间隔应与 Chainlink 配置一致
     *      - 精度校正因子与 Chainlink 返回精度相关
     */
    function run() external {
        uint256 deployerPrivateKey = vm.envUint("MetaNode_DEPLOYER_PK");
        vm.startBroadcast(deployerPrivateKey);
        new OracleAdaptor(
            // chainlink: 资产/USD 价格源
            0x64c911996D3c6aC71f9b455B1E8E7266BcbD848F,
            // decimalsCorrection: 精度校正（10 表示 Chainlink 返回 10 位小数）
            10,
            // heartbeatInterval: 资产价格心跳（86400秒 = 24小时）
            86_400,
            // usdcHeartbeat: USDC 价格心跳
            86_400,
            // usdcSource: USDC/USD 价格源
            0x7e860098F58bBFC8648a4311b374B1D669a2bc6B,
            // priceThreshold: 价格偏离阈值（5e16 = 5%）
            5e16
        );
        vm.stopBroadcast();
    }
}
