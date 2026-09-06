/*
    Copyright 2022 MetaNode Protocol
    SPDX-License-Identifier: BUSL-1.1
*/

pragma solidity ^0.8.19;

/**
 * @title IPerpetual - 永续合约接口
 * @notice Perpetual 合约的接口定义
 * 
 * 永续合约是单个交易市场（如 BTC-PERP）的核心合约
 * 负责记录交易者的仓位和处理交易/清算
 * 
 * 核心概念：
 * - paper: 仓位数量（正=多头，负=空头）
 * - credit: 资金量，与 paper 配合计算盈亏
 */
interface IPerpetual {
    /**
     * @notice 查询交易者的仓位余额
     * @param trader 交易者地址
     * @return paper 仓位数量
     *         正数表示多头，负数表示空头
     * @return credit 资金量
     *         不直接反映仓位方向或入场价格
     *         仅用于计算风险比率和净值
     */
    function balanceOf(address trader) external view returns (int256 paper, int256 credit);

    /**
     * @notice 执行交易
     * @param tradeData 编码的交易数据
     * @dev 交易数据会转发给 Dealer 合约进行验证和撮合
     *      然后 Perpetual 合约执行结算
     */
    function trade(bytes calldata tradeData) external;

    /**
     * @notice 清算仓位
     * @param liquidator 清算者地址
     * @param liquidatedTrader 被清算者地址
     * @param requestPaper 请求清算的仓位数量
     *        正数表示清算多头，负数表示清算空头
     * @param expectCredit 期望的 credit 变化（价格保护）
     *        清算多头时期望收到的金额，清算空头时期望支付的金额
     * @return liqtorPaperChange 清算者最终的 paper 变化
     * @return liqtorCreditChange 清算者最终的 credit 变化
     * 
     * @dev 清算是公开的，任何人都可以清算不安全的仓位
     *      清算可能不会执行或部分执行，原因包括：
     *      1. 其他人先提交了清算请求
     *      2. 交易者及时补充了保证金
     *      3. 标记价格变化超出了价格保护范围
     * 
     *      清算数量会被限制在实际仓位大小内
     *      例如：仓位剩余 10 ETH，请求清算 15 ETH，只会执行 10 ETH
     */
    function liquidate(
        address liquidator,
        address liquidatedTrader,
        int256 requestPaper,
        int256 expectCredit
    )
        external
        returns (int256 liqtorPaperChange, int256 liqtorCreditChange);

    /**
     * @notice 获取当前资金费率
     * @return 资金费率（1e18 精度，累计值）
     * @dev 资金费率是累计值，其变化量才是实际的费率
     *      正费率：多头支付给空头
     *      负费率：空头支付给多头
     */
    function getFundingRate() external view returns (int256);

    /**
     * @notice 更新资金费率
     * @param newFundingRate 新的资金费率
     * @dev 仅 owner（Dealer 合约）可调用
     */
    function updateFundingRate(int256 newFundingRate) external;
}
