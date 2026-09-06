/*
    Copyright 2022 MetaNode Protocol
    SPDX-License-Identifier: Apache-2.0
*/

pragma solidity ^0.8.9;

import "forge-std/Script.sol";
import "../src/oracle/EmergencyOracle.sol";

/**
 * @title EmergencyOracleScript - 部署应急预言机
 * @notice 用于部署应急备用预言机
 * 
 * 部署命令：
 * forge script script/deployEmergencyOracle.s.sol:EmergencyOracleScript --rpc-url <RPC_URL> --broadcast
 * 
 * 环境变量：
 * - MetaNode_DEPLOYER_PK: 部署者私钥
 * 
 * 使用场景：
 * - 主预言机（Chainlink）故障时的备用
 * - 新资产上线时主预言机尚未支持
 * - 需要手动设定价格的特殊场景
 * 
 * 安全注意事项：
 * - 应急预言机应仅作为临时方案
 * - 价格更新需要管理员权限
 * - 建议设置价格后及时切换回主预言机
 */
contract EmergencyOracleScript is Script {
    // 排除在覆盖率报告之外
    function test() public { }

    /**
     * @notice 执行部署
     * @dev 部署后需要：
     *      1. 调用 setMarkPrice 设置初始价格
     *      2. 在 Dealer 的 setPerpRiskParams 中指定为 markPriceSource
     */
    function run() public {
        uint256 deployerPrivateKey = vm.envUint("MetaNode_DEPLOYER_PK");
        vm.startBroadcast(deployerPrivateKey);
        // 部署应急预言机，描述用于标识用途
        new EmergencyOracle("WUSDM/USDC");
        vm.stopBroadcast();
    }
}
