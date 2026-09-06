// SPDX-License-Identifier: GPL-2.0-or-later

pragma solidity ^0.8.19;

import "../lib/forge-std/src/Script.sol";
import "../src/subaccount/SubaccountFactory.sol";

/**
 * @title SubaccountFactoryScript - 部署子账户工厂合约
 * @notice 用于部署 SubaccountFactory 合约，允许用户创建子账户
 * 
 * 部署命令：
 * forge script script/deployFactory.s.sol:SubaccountFactoryScript --rpc-url <RPC_URL> --broadcast
 * 
 * 环境变量：
 * - MetaNode_DEPLOYER_PK: 部署者私钥
 * 
 * 子账户功能：
 * - 隔离仓位风险
 * - 授权操作员代为交易
 * - 使用 Clone 模式降低部署成本
 */
contract SubaccountFactoryScript is Script {
    // 排除在覆盖率报告之外
    function test() public { }

    /**
     * @notice 执行部署
     * @dev 工厂合约部署后即可使用
     *      用户通过调用 newSubaccount() 创建子账户
     */
    function run() external {
        uint256 deployerPrivateKey = vm.envUint("MetaNode_DEPLOYER_PK");
        vm.startBroadcast(deployerPrivateKey);
        new SubaccountFactory();
        vm.stopBroadcast();
    }
}
