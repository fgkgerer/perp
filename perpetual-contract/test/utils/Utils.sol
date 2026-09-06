// SPDX-License-Identifier: BUSL-1.1

pragma solidity >=0.8.0;

import { Test } from "forge-std/Test.sol";

/**
 * @title Utils - 测试工具合约
 * @notice 提供测试中常用的工具函数
 * 
 * 功能：
 * 1. 创建测试用户
 * 2. 操作区块链状态（区块号、时间等）
 */
contract Utils is Test {
    // 排除在覆盖率报告之外
    function test() public { }

    /// @notice 用于生成下一个用户地址的种子
    bytes32 internal nextUser = keccak256(abi.encodePacked("user address"));

    /**
     * @notice 获取下一个唯一用户地址
     * @return 新的 payable 用户地址
     * @dev 每次调用生成一个新的确定性地址
     */
    function getNextUserAddress() external returns (address payable) {
        address payable user = payable(address(uint160(uint256(nextUser))));
        nextUser = keccak256(abi.encodePacked(nextUser));
        return user;
    }

    /**
     * @notice 批量创建测试用户
     * @param userNum 要创建的用户数量
     * @return users 用户地址数组
     * @dev 每个用户会获得 100 ETH 的初始余额
     */
    function createUsers(uint256 userNum) external returns (address payable[] memory) {
        address payable[] memory users = new address payable[](userNum);
        for (uint256 i = 0; i < userNum; i++) {
            address payable user = this.getNextUserAddress();
            // 给每个用户 100 ETH
            vm.deal(user, 100 ether);
            users[i] = user;
        }

        return users;
    }

    /**
     * @notice 快进区块号
     * @param numBlocks 要快进的区块数
     * @dev 用于测试依赖区块号的逻辑
     */
    function mineBlocks(uint256 numBlocks) external {
        uint256 targetBlock = block.number + numBlocks;
        vm.roll(targetBlock);
    }
}
