/*
    Copyright 2022 MetaNode Protocol
    SPDX-License-Identifier: BUSL-1.1
*/

pragma solidity ^0.8.19;

import "@openzeppelin/contracts/interfaces/IERC1271.sol";
import "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";
import "@openzeppelin/contracts/utils/Address.sol";
import "./interfaces/IDealer.sol";
import "./libraries/Errors.sol";
import "./libraries/Funding.sol";
import "./libraries/Liquidation.sol";
import "./libraries/Operation.sol";
import "./libraries/Position.sol";
import "./libraries/SignedDecimalMath.sol";
import "./libraries/Trading.sol";
import "./MetaNodeStorage.sol";

/**
 * @title MetaNodeExternal - 永续合约系统外部调用函数
 * @notice 包含用户可调用的核心业务函数：存款、取款、交易、清算等
 * 
 * 核心业务流程：
 * 1. 资金流程：deposit -> requestWithdraw -> executeWithdraw
 * 2. 交易流程：用户签名订单 -> orderSender 调用 trade -> approveTrade 验证和结算
 * 3. 清算流程：requestLiquidation -> 计算清算金额 -> 结算仓位
 */
abstract contract MetaNodeExternal is MetaNodeStorage, IDealer {
    using SignedDecimalMath for int256;
    using SafeERC20 for IERC20;

    // ==================== 资金相关函数 ====================

    /**
     * @notice 存入保证金
     * @param primaryAmount 主资产（USDC）数量
     * @param secondaryAmount 次级资产数量（如果有）
     * @param to 存入的目标账户地址
     * @dev 用户将资产从钱包转入交易系统，获得交易所需的保证金
     */
    function deposit(uint256 primaryAmount, uint256 secondaryAmount, address to) external nonReentrant {
        Funding.deposit(state, primaryAmount, secondaryAmount, to);
    }

    /**
     * @notice 请求取款（第一步）
     * @param from 取款来源账户
     * @param primaryAmount 主资产取款数量
     * @param secondaryAmount 次级资产取款数量
     * @dev 取款分两步：先请求，等待时间锁后再执行
     *      时间锁设计是为了防止在订单成交前取走保证金
     */
    function requestWithdraw(address from, uint256 primaryAmount, uint256 secondaryAmount) external nonReentrant {
        Funding.requestWithdraw(state, from, primaryAmount, secondaryAmount);
    }

    /**
     * @notice 执行取款（第二步）
     * @param from 取款来源账户
     * @param to 资金接收地址
     * @param isInternal 是否为内部转账（不转出合约，只转给另一个账户）
     * @param param 回调参数
     * @dev 在时间锁到期后执行实际的资金转出
     */
    function executeWithdraw(address from, address to, bool isInternal, bytes memory param) external nonReentrant {
        Funding.executeWithdraw(state, from, to, isInternal, param);
    }

    /**
     * @notice 快速取款（一步完成）
     * @param from 取款来源账户
     * @param to 资金接收地址
     * @param primaryAmount 主资产取款数量
     * @param secondaryAmount 次级资产取款数量
     * @param isInternal 是否为内部转账
     * @param param 回调参数
     * @dev 当 fastWithdraw 功能开启时，可以跳过时间锁直接取款
     *      仅限白名单用户或特定条件下使用
     */
    function fastWithdraw(
        address from,
        address to,
        uint256 primaryAmount,
        uint256 secondaryAmount,
        bool isInternal,
        bytes memory param
    )
        external
        nonReentrant
    {
        Funding.fastWithdraw(state, from, to, primaryAmount, secondaryAmount, isInternal, param);
    }

    /**
     * @notice 设置操作员
     * @param operator 操作员地址
     * @param isValid 是否授权
     * @dev 操作员可以代替用户签名订单，但不能操作资金
     *      常用于机器人交易或子账户管理
     */
    function setOperator(address operator, bool isValid) external {
        Operation.setOperator(state, msg.sender, operator, isValid);
    }

    /**
     * @notice 授权资金操作员
     * @param operator 操作员地址
     * @param primaryAmount 授权的主资产额度
     * @param secondaryAmount 授权的次级资产额度
     * @dev 允许操作员在授权额度内操作用户资金
     */
    function approveFundOperator(address operator, uint256 primaryAmount, uint256 secondaryAmount) external {
        Operation.approveFundOperator(state, msg.sender, operator, primaryAmount, secondaryAmount);
    }

    /**
     * @notice 处理坏账
     * @param liquidatedTrader 被清算交易者地址
     * @dev 当交易者仓位全部清算后仍有负债时，由保险基金承担
     */
    function handleBadDebt(address liquidatedTrader) external {
        Liquidation.handleBadDebt(state, liquidatedTrader);
    }

    // ==================== 仅限已注册永续合约调用 ====================

    /**
     * @notice 请求清算（由 Perpetual 合约调用）
     * @param executor 执行清算的地址
     * @param liquidator 清算者（接手仓位的人）
     * @param liquidatedTrader 被清算者
     * @param requestPaperAmount 请求清算的仓位数量
     * @return liqtorPaperChange 清算者的 paper 变化
     * @return liqtorCreditChange 清算者的 credit 变化
     * @return liqedPaperChange 被清算者的 paper 变化
     * @return liqedCreditChange 被清算者的 credit 变化
     * @dev 清算机制：当交易者保证金率低于维持保证金率时，可被清算
     *      清算者以一定折扣接手被清算者的仓位
     */
    function requestLiquidation(
        address executor,
        address liquidator,
        address liquidatedTrader,
        int256 requestPaperAmount
    )
        external
        onlyRegisteredPerp
        returns (int256 liqtorPaperChange, int256 liqtorCreditChange, int256 liqedPaperChange, int256 liqedCreditChange)
    {
        return Liquidation.requestLiquidation(
            state, msg.sender, executor, liquidator, liquidatedTrader, requestPaperAmount
        );
    }

    /**
     * @notice 开仓通知（由 Perpetual 合约调用）
     * @param trader 交易者地址
     * @dev 当交易者在某个市场首次开仓时调用
     *      用于跟踪用户的持仓市场列表
     */
    function openPosition(address trader) external onlyRegisteredPerp {
        Position._openPosition(state, trader);
    }

    /**
     * @notice 实现盈亏结算（由 Perpetual 合约调用）
     * @param trader 交易者地址
     * @param pnl 盈亏金额
     * @dev 当交易者平仓时调用，将实现盈亏计入账户余额
     */
    function realizePnl(address trader, int256 pnl) external onlyRegisteredPerp {
        Position._realizePnl(state, trader, pnl);
    }

    /**
     * @notice 批准交易（核心撮合函数，由 Perpetual 合约调用）
     * @param orderSender 订单发送者地址（撮合引擎）
     * @param tradeData 编码的交易数据，包含订单列表、签名、成交数量
     * @return traderList 参与交易的交易者列表
     * @return paperChangeList 各交易者的 paper（仓位）变化
     * @return creditChangeList 各交易者的 credit（资金）变化
     * 
     * @dev 核心交易流程：
     * 1. 解码交易数据，获取订单列表、签名、成交数量
     * 2. 验证每个订单：
     *    - 签名验证（支持 EOA 签名和合约签名 EIP-1271）
     *    - 订单未过期
     *    - 订单价格有效（paper 和 credit 异号）
     *    - 订单未超额成交
     *    - 防止自成交
     * 3. 撮合订单，计算各方的 paper 和 credit 变化
     * 4. 收取手续费给 orderSender
     * 5. 检查 orderSender 安全性（如果需要支付手续费）
     */
    function approveTrade(
        address orderSender,
        bytes calldata tradeData
    )
        external
        onlyRegisteredPerp
        returns (
            address[] memory, // 交易者列表
            int256[] memory,  // paper（仓位数量）变化列表
            int256[] memory   // credit（资金）变化列表
        )
    {
        // 验证订单发送者是否为授权的撮合引擎
        require(state.validOrderSender[orderSender], Errors.INVALID_ORDER_SENDER);

        /*
            解析交易数据
            传入所有需要撮合的订单及其签名
            以及每个订单要成交的数量
        */
        (Types.Order[] memory orderList, bytes[] memory signatureList, uint256[] memory matchPaperAmount) =
            abi.decode(tradeData, (Types.Order[], bytes[], uint256[]));
        bytes32[] memory orderHashList = new bytes32[](orderList.length);

        // 验证所有订单
        for (uint256 i = 0; i < orderList.length;) {
            Types.Order memory order = orderList[i];
            // 计算订单哈希（用于签名验证和订单追踪）
            bytes32 orderHash = EIP712._hashTypedDataV4(domainSeparator, Trading._structHash(order));
            orderHashList[i] = orderHash;
            
            // 验证签名
            (address recoverSigner,) = ECDSA.tryRecover(orderHash, signatureList[i]);
            // 签名者必须是订单所有者或其授权的操作员
            if (recoverSigner != order.signer && !state.operatorRegistry[order.signer][recoverSigner]) {
                // 如果签名者是合约，使用 EIP-1271 标准验证
                if (Address.isContract(order.signer)) {
                    require(
                        IERC1271(order.signer).isValidSignature(orderHash, signatureList[i]) == 0x1626ba7e,
                        Errors.INVALID_ORDER_SIGNATURE
                    );
                } else {
                    revert(Errors.INVALID_ORDER_SIGNATURE);
                }
            }
            
            // 验证订单基本要求
            require(Trading._info2Expiration(order.info) >= block.timestamp, Errors.ORDER_EXPIRED);
            // paper 和 credit 必须异号（买单：paper>0, credit<0；卖单：paper<0, credit>0）
            require(
                (order.paperAmount < 0 && order.creditAmount > 0) || (order.paperAmount > 0 && order.creditAmount < 0),
                Errors.ORDER_PRICE_NEGATIVE
            );
            // 订单必须属于当前调用的永续合约
            require(order.perp == msg.sender, Errors.PERP_MISMATCH);
            // 防止自成交（第一个订单的签名者不能与后续订单相同）
            require(i == 0 || order.signer != orderList[0].signer, Errors.ORDER_SELF_MATCH);
            
            // 更新订单已成交数量
            state.orderFilledPaperAmount[orderHash] += matchPaperAmount[i];
            // 检查是否超额成交
            require(
                state.orderFilledPaperAmount[orderHash] <= int256(orderList[i].paperAmount).abs(),
                Errors.ORDER_FILLED_OVERFLOW
            );
            unchecked {
                ++i;
            }
        }

        // 执行订单撮合，计算各方变化
        Types.MatchResult memory result = Trading._matchOrders(state, orderHashList, orderList, matchPaperAmount);

        // 收取手续费给订单发送者（撮合引擎）
        state.primaryCredit[orderSender] += result.orderSenderFee;
        // 如果订单发送者需要支付手续费（负数），检查其账户安全性
        if (result.orderSenderFee < 0) {
            require(Liquidation._isSolidIMSafe(state, orderSender), Errors.ORDER_SENDER_NOT_SAFE);
        }

        return (result.traderList, result.paperChangeList, result.creditChangeList);
    }
}
