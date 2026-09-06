// SPDX-License-Identifier: GPL-2.0-or-later

pragma solidity ^0.8.19;

import "../lib/forge-std/src/Script.sol";
import "../src/MetaNodeDealer.sol";

/**
 * @title MetaNodeDealerScript - 部署 MetaNodeDealer 合约
 * @notice 用于部署永续合约交易系统的核心 Dealer 合约
 * 
 * 部署命令：
 * forge script script/deployMetaNodeDealer.s.sol:MetaNodeDealerScript --rpc-url <RPC_URL> --broadcast
 * 
 * 环境变量：
 * - MetaNode_DEPLOYER_PK: 部署者私钥
 * 
 * 部署参数：
 * - primaryAsset: 主资产地址（通常是 USDC）
 */
contract MetaNodeDealerScript is Script {
    // 排除在覆盖率报告之外
    function test() public { }

    /**
     * @notice 执行部署
     * @dev 部署后需要进行以下配置：
     *      1. setInsurance: 设置保险账户
     *      2. setFundingRateKeeper: 设置资金费率更新者
     *      3. setOrderSender: 设置撮合引擎地址
     *      4. setMaxPositionAmount: 设置最大持仓数
     *      5. setWithdrawTimeLock: 设置取款时间锁
     *      6. setPerpRiskParams: 注册永续合约市场
     */
    function run() external {
        uint256 deployerPrivateKey = vm.envUint("MetaNode_DEPLOYER_PK");
        vm.startBroadcast(deployerPrivateKey);
        // 部署 MetaNodeDealer，传入主资产（USDC）地址
        new MetaNodeDealer(0x834D14F87700e5fFc084e732c7381673133cdbcC);
        vm.stopBroadcast();
    }
}
