/*
    Copyright 2022 MetaNode Protocol
    SPDX-License-Identifier: BUSL-1.1
*/

pragma solidity ^0.8.19;

import "./MetaNodeExternal.sol";
import "./MetaNodeOperation.sol";
import "./MetaNodeView.sol";

/**
 * @title MetaNodeDealer - 永续合约交易系统主入口
 * @notice 这是整个永续合约交易系统的顶层入口合约
 * 
 * 架构说明：
 * - MetaNodeView: 视图函数（查询账户状态、风险参数等）
 * - MetaNodeExternal: 外部调用函数（存款、取款、交易、清算等）
 * - MetaNodeOperation: 管理员函数（设置参数、注册市场等）
 * - MetaNodeStorage: 数据存储层（所有状态变量）
 * 
 * 核心功能：
 * 1. 保证金管理：存款、取款、资金划转
 * 2. 订单撮合：验证订单签名、执行交易结算
 * 3. 风险管理：保证金率检查、强制清算
 * 4. 资金费率：定期更新资金费率
 */
contract MetaNodeDealer is MetaNodeExternal, MetaNodeOperation, MetaNodeView {
    /**
     * @notice 构造函数
     * @param _primaryAsset 主资产地址（通常是 USDC）
     * @dev 主资产用于保证金存款和结算
     */
    constructor(address _primaryAsset) MetaNodeStorage() {
        state.primaryAsset = _primaryAsset;
    }

    /**
     * @notice 返回合约版本号
     * @return 版本字符串
     */
    function version() external pure returns (string memory) {
        return "MetaNodeDealer V1.1";
    }
}
