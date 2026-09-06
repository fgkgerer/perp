/*
    Copyright 2022 MetaNode Protocol
    SPDX-License-Identifier: BUSL-1.1
*/

pragma solidity ^0.8.19;

import "../init/TradingInit.sol";
import "../utils/Checkers.sol";
import "../../src/libraries/Types.sol";
import "../mocks/MockERC1271.sol";
import "../mocks/MockERC1271Failed.sol";

/**
 * @title ScenarioSignatureTest - 订单签名场景测试
 * @author MetaNode Team
 * @notice 演示 EIP-712 订单签名的完整流程
 * 
 * ============================================================
 *                      EIP-712 签名说明
 * ============================================================
 * 
 * EIP-712 是结构化数据签名标准，提供：
 * 1. 人类可读的签名内容
 * 2. 防重放攻击
 * 3. 域分隔符防止跨合约重放
 * 
 * 订单签名流程：
 * 1. 构建 Types.Order 结构体
 * 2. 计算域分隔符 (domainSeparator)
 * 3. 计算结构体哈希 (structHash)
 * 4. 计算最终消息哈希
 * 5. 使用私钥签名
 * 
 * ============================================================
 *                   前端开发参考
 * ============================================================
 * 
 * JavaScript 签名示例：
 * 
 * const domain = {
 *   name: "MetaNode",
 *   version: "1",
 *   chainId: chainId,
 *   verifyingContract: dealerAddress
 * };
 * 
 * const types = {
 *   Order: [
 *     { name: "perp", type: "address" },
 *     { name: "signer", type: "address" },
 *     { name: "paperAmount", type: "int128" },
 *     { name: "creditAmount", type: "int128" },
 *     { name: "info", type: "bytes32" }
 *   ]
 * };
 * 
 * const order = {
 *   perp: perpAddress,
 *   signer: userAddress,
 *   paperAmount: 1e18,
 *   creditAmount: -30000e6,
 *   info: infoBytes32
 * };
 * 
 * const signature = await signer._signTypedData(domain, types, order);
 */
contract ScenarioSignatureTest is Checkers {
    /**
     * @notice 测试前准备
     */
    function setUp() public override {
        super.setUp();
        
        vm.prank(traders[0]);
        metaNodeDealer.deposit(50_000e6, 0, traders[0]);
        vm.prank(traders[1]);
        metaNodeDealer.deposit(50_000e6, 0, traders[1]);
    }
    
    // ==================== 场景1: 基础 EOA 签名 ====================
    
    /**
     * @notice 场景1：EOA（外部账户）订单签名
     * @dev 演示完整的签名构建和验证流程
     * 
     * 签名流程：
     * 1. 构建订单结构体
     * 2. 计算 EIP-712 域分隔符
     * 3. 计算消息哈希
     * 4. 使用私钥签名
     */
    function testScenario1_EOASignature() public {
        // 使用辅助函数构建订单并签名
        (Types.Order memory order1, bytes memory sig1) = 
            buildOrder(traders[0], tradersKey[0], 1e18, -30_000e6, address(perpList[0]));
        
        (Types.Order memory order2, bytes memory sig2) = 
            buildOrder(traders[1], tradersKey[1], -1e18, 30_000e6, address(perpList[0]));
        
        // 验证签名长度
        assertEq(sig1.length, 65, "Signature should be 65 bytes");
        
        // 构建交易数据
        Types.Order[] memory orders = new Types.Order[](2);
        orders[0] = order1;
        orders[1] = order2;
        
        bytes[] memory sigs = new bytes[](2);
        sigs[0] = sig1;
        sigs[1] = sig2;
        
        uint256[] memory amounts = new uint256[](2);
        amounts[0] = 1e18;
        amounts[1] = 1e18;
        
        // 执行交易
        perpList[0].trade(abi.encode(orders, sigs, amounts));
        
        // 验证仓位
        (int256 paperResult,) = perpList[0].balanceOf(traders[0]);
        assertEq(paperResult, 1e18, "Position should be opened");
    }
    
    // ==================== 场景2: 订单过期 ====================
    
    /**
     * @notice 场景2：过期订单被拒绝
     * @dev 验证订单过期检查
     */
    function testScenario2_ExpiredOrder() public {
        // 构建过期订单（过期时间设为过去）
        int64 makerFeeRate = 1e14;
        int64 takerFeeRate = 5e14;
        uint64 expiration = uint64(block.timestamp - 1);  // 已过期
        uint64 nonce = uint64(block.timestamp);
        
        bytes memory infoBytes = abi.encodePacked(makerFeeRate, takerFeeRate, expiration, nonce);
        
        Types.Order memory order = Types.Order({
            perp: address(perpList[0]),
            signer: traders[0],
            paperAmount: 1e18,
            creditAmount: -30_000e6,
            info: bytes32(infoBytes)
        });
        
        // 签名
        bytes32 domainSeparator = EIP712Test._buildDomainSeparator("MetaNode", "1", address(metaNodeDealer));
        bytes32 digest = keccak256(abi.encodePacked("\x19\x01", domainSeparator, EIP712Test._structHash(order)));
        (uint8 v, bytes32 r, bytes32 s) = vm.sign(tradersKey[0], digest);
        bytes memory signature = abi.encodePacked(r, s, v);
        
        // 构建对手盘
        (Types.Order memory order2, bytes memory sig2) = 
            buildOrder(traders[1], tradersKey[1], -1e18, 30_000e6, address(perpList[0]));
        
        Types.Order[] memory orders = new Types.Order[](2);
        orders[0] = order;
        orders[1] = order2;
        
        bytes[] memory sigs = new bytes[](2);
        sigs[0] = signature;
        sigs[1] = sig2;
        
        uint256[] memory amounts = new uint256[](2);
        amounts[0] = 1e18;
        amounts[1] = 1e18;
        
        // 应该失败
        vm.expectRevert("META_ORDER_EXPIRED");
        perpList[0].trade(abi.encode(orders, sigs, amounts));
    }
    
    // ==================== 场景3: 订单部分成交 ====================
    
    /**
     * @notice 场景3：订单部分成交后填充量检查
     * @dev 验证 filledAmount 跟踪
     */
    function testScenario3_PartialFill() public {
        // 构建大订单（做多 3 BTC）
        (Types.Order memory bigOrder, bytes memory bigSig) = 
            buildOrder(traders[0], tradersKey[0], 3e18, -90_000e6, address(perpList[0]));
        
        // 第一次成交 1 BTC
        (Types.Order memory order1, bytes memory sig1) = 
            buildOrder(traders[1], tradersKey[1], -1e18, 30_000e6, address(perpList[0]));
        
        Types.Order[] memory orders = new Types.Order[](2);
        orders[0] = bigOrder;
        orders[1] = order1;
        
        bytes[] memory sigs = new bytes[](2);
        sigs[0] = bigSig;
        sigs[1] = sig1;
        
        uint256[] memory amounts = new uint256[](2);
        amounts[0] = 1e18;  // 只成交 1 BTC
        amounts[1] = 1e18;
        
        perpList[0].trade(abi.encode(orders, sigs, amounts));
        
        // 查询订单已成交量
        bytes32 orderHash = keccak256(abi.encodePacked("\x19\x01", 
            EIP712Test._buildDomainSeparator("MetaNode", "1", address(metaNodeDealer)), 
            EIP712Test._structHash(bigOrder)));
        uint256 filledAmount = metaNodeDealer.getOrderFilledAmount(orderHash);
        assertEq(filledAmount, 1e18, "Should have 1 BTC filled");
        
        // 第二次成交 2 BTC
        (Types.Order memory order2, bytes memory sig2) = 
            buildOrder(traders[1], tradersKey[1], -2e18, 60_000e6, address(perpList[0]));
        
        orders[1] = order2;
        sigs[1] = sig2;
        amounts[0] = 2e18;  // 再成交 2 BTC
        amounts[1] = 2e18;
        
        perpList[0].trade(abi.encode(orders, sigs, amounts));
        
        // 再次查询
        filledAmount = metaNodeDealer.getOrderFilledAmount(orderHash);
        assertEq(filledAmount, 3e18, "Should have 3 BTC total filled");
        
        // 验证仓位
        (int256 paper,) = perpList[0].balanceOf(traders[0]);
        assertEq(paper, 3e18, "Position should be 3 BTC");
    }
    
    // ==================== 场景4: 智能合约钱包签名 ====================
    
    /**
     * @notice 场景4：ERC-1271 智能合约钱包签名
     * @dev 演示智能合约作为交易者
     */
    function testScenario4_ContractWalletSignature() public {
        // 部署模拟智能合约钱包
        MockERC1271 contractWallet = new MockERC1271();
        
        // 为合约钱包存入保证金
        usdc.mint(address(contractWallet), 50_000e6);
        vm.startPrank(address(contractWallet));
        usdc.approve(address(metaNodeDealer), 50_000e6);
        metaNodeDealer.deposit(50_000e6, 0, address(contractWallet));
        vm.stopPrank();
        
        // 构建合约钱包的订单
        Types.Order memory walletOrder = Types.Order({
            perp: address(perpList[0]),
            signer: address(contractWallet),  // 签名者是合约
            paperAmount: 1e18,
            creditAmount: -30_000e6,
            info: bytes32(abi.encodePacked(int64(1e14), int64(5e14), uint64(block.timestamp), uint64(block.timestamp)))
        });
        
        // 对于合约钱包，签名可以是任意内容，只要 isValidSignature 返回正确
        bytes memory walletSig = "contract_wallet_signature";
        
        // 构建对手盘
        (Types.Order memory order2, bytes memory sig2) = 
            buildOrder(traders[1], tradersKey[1], -1e18, 30_000e6, address(perpList[0]));
        
        Types.Order[] memory orders = new Types.Order[](2);
        orders[0] = walletOrder;
        orders[1] = order2;
        
        bytes[] memory sigs = new bytes[](2);
        sigs[0] = walletSig;
        sigs[1] = sig2;
        
        uint256[] memory amounts = new uint256[](2);
        amounts[0] = 1e18;
        amounts[1] = 1e18;
        
        // 执行交易
        perpList[0].trade(abi.encode(orders, sigs, amounts));
        
        // 验证仓位
        (int256 paper,) = perpList[0].balanceOf(address(contractWallet));
        assertEq(paper, 1e18, "Contract wallet should have position");
    }
    
    // ==================== 场景5: 无效签名被拒绝 ====================
    
    /**
     * @notice 场景5：无效签名被拒绝
     * @dev 验证签名验证机制
     */
    function testScenario5_InvalidSignatureRejected() public {
        // 构建订单
        (Types.Order memory order, bytes memory validSig) = 
            buildOrder(traders[0], tradersKey[0], 1e18, -30_000e6, address(perpList[0]));
        
        // 篡改签名
        bytes memory invalidSig = validSig;
        invalidSig[0] = bytes1(uint8(invalidSig[0]) + 1);  // 修改第一个字节
        
        (Types.Order memory order2, bytes memory sig2) = 
            buildOrder(traders[1], tradersKey[1], -1e18, 30_000e6, address(perpList[0]));
        
        Types.Order[] memory orders = new Types.Order[](2);
        orders[0] = order;
        orders[1] = order2;
        
        bytes[] memory sigs = new bytes[](2);
        sigs[0] = invalidSig;  // 使用无效签名
        sigs[1] = sig2;
        
        uint256[] memory amounts = new uint256[](2);
        amounts[0] = 1e18;
        amounts[1] = 1e18;
        
        // 应该失败
        vm.expectRevert("META_INVALID_ORDER_SIGNATURE");
        perpList[0].trade(abi.encode(orders, sigs, amounts));
    }
}
