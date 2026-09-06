/*
    Copyright 2022 MetaNode Protocol
    SPDX-License-Identifier: BUSL-1.1
*/

pragma solidity ^0.8.19;

import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/utils/math/SafeCast.sol";

/**
 * @title EmergencyOracle - 应急预言机
 * @notice 当第三方预言机不可用时的应急备用方案
 * 
 * 使用场景：
 * 1. Chainlink 等主预言机故障时的备用
 * 2. 新市场上线时主预言机尚未支持
 * 3. 紧急情况下需要手动设定价格
 * 
 * 安全考虑：
 * - 仅管理员可更新价格
 * - 价格更新会记录轮次和时间戳
 * - 应仅作为临时方案使用
 * 
 * @dev 继承 Ownable，使用 OpenZeppelin 的访问控制
 */
contract EmergencyOracle is Ownable {
    /// @notice 当前价格（1e18 精度）
    uint256 public price;
    
    /// @notice 价格更新轮次
    uint256 public roundId;
    
    /// @notice 预言机描述（如 "BTC/USD Emergency Oracle"）
    string public description;

    /// @notice 价格更新事件（与 Chainlink 格式对齐）
    event AnswerUpdated(int256 indexed current, uint256 indexed roundId, uint256 updatedAt);

    /**
     * @notice 构造函数
     * @param _description 预言机描述
     */
    constructor(string memory _description) Ownable() {
        description = _description;
    }

    /**
     * @notice 获取标记价格
     * @return 当前价格（1e18 精度）
     * @dev 如果价格未设置（为 0），调用方需自行处理
     */
    function getMarkPrice() external view returns (uint256) {
        return price;
    }

    /**
     * @notice 获取资产价格
     * @return 当前价格（1e18 精度）
     */
    function getAssetPrice() external view returns (uint256) {
        return price;
    }

    /**
     * @notice 设置新价格
     * @param newPrice 新价格（1e18 精度）
     * @dev 仅管理员可调用
     *      每次更新会增加 roundId 并记录时间戳
     */
    function setMarkPrice(uint256 newPrice) external onlyOwner {
        price = newPrice;
        emit AnswerUpdated(SafeCast.toInt256(price), roundId, block.timestamp);
        roundId += 1;
    }
}
