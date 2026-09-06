// SPDX-License-Identifier: GPL-2.0-or-later

pragma solidity ^0.8.19;

import "../lib/forge-std/src/Script.sol";
import "../src/Perpetual.sol";

/**
 * @title PerpetualScript - 部署永续合约市场
 * @notice 用于部署单个永续合约市场（如 BTC-PERP、ETH-PERP）
 * 
 * 部署命令：
 * forge script script/deployPerpetual.s.sol:PerpetualScript --rpc-url <RPC_URL> --broadcast
 * 
 * 环境变量：
 * - MetaNode_DEPLOYER_PK: 部署者私钥
 * 
 * 部署参数：
 * - owner: MetaNodeDealer 合约地址
 * 
 * 部署后配置：
 * 部署完成后，需要在 MetaNodeDealer 中注册该市场：
 * dealer.setPerpRiskParams(perpAddress, riskParams);
 */
contract PerpetualScript is Script {
    // 排除在覆盖率报告之外
    function test() public { }

    /**
     * @notice 执行部署
     * @dev 注意：
     *      1. owner 必须是已部署的 MetaNodeDealer 地址
     *      2. 部署后需要在 Dealer 中注册并设置风险参数
     *      3. 需要配置价格预言机
     */
    function run() external {
        uint256 deployerPrivateKey = vm.envUint("MetaNode_DEPLOYER_PK");
        vm.startBroadcast(deployerPrivateKey);
        // 部署永续合约，owner 设为 MetaNodeDealer 地址
        new Perpetual(0x2f7c3cF9D9280B165981311B822BecC4E05Fe635);
        vm.stopBroadcast();
    }
}
