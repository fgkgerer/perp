// SPDX-License-Identifier: UNLICENSED

pragma solidity ^0.8.9;

import "forge-std/Script.sol";
import "../src/support/TestERC20.sol";
import "forge-std/Test.sol";

/**
 * @title TokenScript - 部署测试用 ERC20 代币
 * @notice 用于在测试网部署模拟代币
 * 
 * 部署命令：
 * forge script script/deployTestERC20Token.s.sol:TokenScript --rpc-url <RPC_URL> --broadcast
 * 
 * 环境变量：
 * - MetaNode_DEPLOYER_PK: 部署者私钥
 * 
 * 使用场景：
 * - 测试网部署模拟 USDC、WETH 等代币
 * - 本地开发环境测试
 * 
 * TestERC20 特性：
 * - 任何人都可以 mint 代币（仅用于测试）
 * - 可自定义名称、符号、小数位数
 */
contract TokenScript is Script {
    // 排除在覆盖率报告之外
    function test() public { }

    /**
     * @notice 执行部署
     * @dev 可以修改参数部署不同的测试代币：
     *      - TestERC20("USDC", "USDC", 6)  // 模拟 USDC
     *      - TestERC20("WETH", "WETH", 18) // 模拟 WETH
     */
    function run() external {
        uint256 deployerPrivateKey = vm.envUint("MetaNode_DEPLOYER_PK");
        vm.startBroadcast(deployerPrivateKey);
        // 部署测试代币（名称、符号、小数位数）
        new TestERC20("ARB", "ARB", 18);
        console2.log("deploy ARB");
        vm.stopBroadcast();
    }
}
