-- MetaNode 数据库表结构
-- 使用 MySQL 8.0+

CREATE DATABASE IF NOT EXISTS metanode DEFAULT CHARSET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE metanode;

-- 订单表
CREATE TABLE IF NOT EXISTS `orders` (
    `id` BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    `order_id` VARCHAR(66) NOT NULL COMMENT '订单ID (hash)',
    `perp` VARCHAR(42) NOT NULL COMMENT '永续合约地址',
    `signer` VARCHAR(42) NOT NULL COMMENT '签名者地址',
    `paper_amount` VARCHAR(78) NOT NULL COMMENT '仓位数量',
    `credit_amount` VARCHAR(78) NOT NULL COMMENT '资金数量',
    `maker_fee_rate` VARCHAR(78) NOT NULL COMMENT 'Maker手续费率',
    `taker_fee_rate` VARCHAR(78) NOT NULL COMMENT 'Taker手续费率',
    `expiration` BIGINT NOT NULL COMMENT '过期时间戳',
    `nonce` BIGINT NOT NULL COMMENT '随机数',
    `signature` TEXT NOT NULL COMMENT 'EIP-712签名',
    `status` TINYINT NOT NULL DEFAULT 0 COMMENT '状态: 0-待处理, 1-部分成交, 2-完全成交, 3-已取消',
    `filled_amount` VARCHAR(78) NOT NULL DEFAULT '0' COMMENT '已成交数量',
    `create_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `update_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY `uk_order_id` (`order_id`),
    INDEX `idx_signer` (`signer`),
    INDEX `idx_perp_status` (`perp`, `status`),
    INDEX `idx_create_time` (`create_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='订单表';

-- 成交记录表
CREATE TABLE IF NOT EXISTS `trades` (
    `id` BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    `trade_id` VARCHAR(66) NOT NULL COMMENT '成交ID',
    `perp` VARCHAR(42) NOT NULL COMMENT '永续合约地址',
    `taker_order_id` VARCHAR(66) NOT NULL COMMENT 'Taker订单ID',
    `maker_order_id` VARCHAR(66) NOT NULL COMMENT 'Maker订单ID',
    `taker` VARCHAR(42) NOT NULL COMMENT 'Taker地址',
    `maker` VARCHAR(42) NOT NULL COMMENT 'Maker地址',
    `paper_amount` VARCHAR(78) NOT NULL COMMENT '成交数量',
    `price` VARCHAR(78) NOT NULL COMMENT '成交价格',
    `taker_fee` VARCHAR(78) DEFAULT '0' COMMENT 'Taker手续费',
    `maker_fee` VARCHAR(78) DEFAULT '0' COMMENT 'Maker手续费',
    `tx_hash` VARCHAR(66) DEFAULT '' COMMENT '链上交易哈希',
    `block_number` BIGINT DEFAULT 0 COMMENT '区块号',
    `create_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY `uk_trade_id` (`trade_id`),
    INDEX `idx_taker` (`taker`),
    INDEX `idx_maker` (`maker`),
    INDEX `idx_perp` (`perp`),
    INDEX `idx_create_time` (`create_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='成交记录表';

-- 仓位表
CREATE TABLE IF NOT EXISTS `positions` (
    `id` BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    `trader` VARCHAR(42) NOT NULL COMMENT '交易者地址',
    `perp` VARCHAR(42) NOT NULL COMMENT '永续合约地址',
    `paper` VARCHAR(78) NOT NULL DEFAULT '0' COMMENT '仓位数量',
    `credit` VARCHAR(78) NOT NULL DEFAULT '0' COMMENT '资金数量',
    `update_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY `uk_trader_perp` (`trader`, `perp`),
    INDEX `idx_trader` (`trader`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='仓位表';

-- 存款记录表
CREATE TABLE IF NOT EXISTS `deposits` (
    `id` BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    `tx_hash` VARCHAR(66) NOT NULL COMMENT '交易哈希',
    `trader` VARCHAR(42) NOT NULL COMMENT '交易者地址',
    `primary_amount` VARCHAR(78) NOT NULL DEFAULT '0' COMMENT '主资产数量',
    `secondary_amount` VARCHAR(78) NOT NULL DEFAULT '0' COMMENT '次级资产数量',
    `block_number` BIGINT NOT NULL COMMENT '区块号',
    `create_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY `uk_tx_hash` (`tx_hash`),
    INDEX `idx_trader` (`trader`),
    INDEX `idx_create_time` (`create_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='存款记录表';

-- 取款记录表
CREATE TABLE IF NOT EXISTS `withdraws` (
    `id` BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    `tx_hash` VARCHAR(66) NOT NULL COMMENT '交易哈希',
    `trader` VARCHAR(42) NOT NULL COMMENT '交易者地址',
    `primary_amount` VARCHAR(78) NOT NULL DEFAULT '0' COMMENT '主资产数量',
    `secondary_amount` VARCHAR(78) NOT NULL DEFAULT '0' COMMENT '次级资产数量',
    `status` TINYINT NOT NULL DEFAULT 0 COMMENT '状态: 0-待处理, 1-已完成, 2-已取消',
    `block_number` BIGINT DEFAULT 0 COMMENT '区块号',
    `create_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY `uk_tx_hash` (`tx_hash`),
    INDEX `idx_trader` (`trader`),
    INDEX `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='取款记录表';

-- 资金费率记录表
CREATE TABLE IF NOT EXISTS `funding_rates` (
    `id` BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    `perp` VARCHAR(42) NOT NULL COMMENT '永续合约地址',
    `rate` VARCHAR(78) NOT NULL COMMENT '资金费率',
    `mark_price` VARCHAR(78) NOT NULL COMMENT '标记价格',
    `index_price` VARCHAR(78) NOT NULL COMMENT '指数价格',
    `settle_time` DATETIME NOT NULL COMMENT '结算时间',
    INDEX `idx_perp` (`perp`),
    INDEX `idx_settle_time` (`settle_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='资金费率记录表';

-- 清算记录表
CREATE TABLE IF NOT EXISTS `liquidations` (
    `id` BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    `tx_hash` VARCHAR(66) NOT NULL COMMENT '交易哈希',
    `perp` VARCHAR(42) NOT NULL COMMENT '永续合约地址',
    `liquidator` VARCHAR(42) NOT NULL COMMENT '清算人地址',
    `liquidated_trader` VARCHAR(42) NOT NULL COMMENT '被清算者地址',
    `paper_change` VARCHAR(78) NOT NULL COMMENT '仓位变化',
    `credit_change` VARCHAR(78) NOT NULL COMMENT '资金变化',
    `insurance_fee` VARCHAR(78) NOT NULL DEFAULT '0' COMMENT '保险费',
    `block_number` BIGINT NOT NULL COMMENT '区块号',
    `create_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY `uk_tx_hash` (`tx_hash`),
    INDEX `idx_liquidator` (`liquidator`),
    INDEX `idx_liquidated_trader` (`liquidated_trader`),
    INDEX `idx_perp` (`perp`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='清算记录表';

-- 市场表
CREATE TABLE IF NOT EXISTS `markets` (
    `id` BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    `address` VARCHAR(42) NOT NULL COMMENT '永续合约地址',
    `name` VARCHAR(32) NOT NULL COMMENT '市场名称',
    `initial_margin_ratio` VARCHAR(78) NOT NULL COMMENT '初始保证金率',
    `liquidation_threshold` VARCHAR(78) NOT NULL COMMENT '清算阈值',
    `liquidation_price_off` VARCHAR(78) NOT NULL COMMENT '清算折扣',
    `insurance_fee_rate` VARCHAR(78) NOT NULL COMMENT '保险费率',
    `is_registered` TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否注册',
    `create_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY `uk_address` (`address`),
    INDEX `idx_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='市场表';

-- K线表
CREATE TABLE IF NOT EXISTS `klines` (
    `id` BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    `perp` VARCHAR(42) NOT NULL COMMENT '永续合约地址',
    `interval_type` VARCHAR(8) NOT NULL COMMENT '周期类型: 1m, 5m, 15m, 1h, 4h, 1d',
    `open_time` DATETIME NOT NULL COMMENT '开盘时间',
    `open_price` VARCHAR(78) NOT NULL COMMENT '开盘价',
    `high_price` VARCHAR(78) NOT NULL COMMENT '最高价',
    `low_price` VARCHAR(78) NOT NULL COMMENT '最低价',
    `close_price` VARCHAR(78) NOT NULL COMMENT '收盘价',
    `volume` VARCHAR(78) NOT NULL DEFAULT '0' COMMENT '成交量',
    UNIQUE KEY `uk_perp_interval_time` (`perp`, `interval_type`, `open_time`),
    INDEX `idx_perp_interval` (`perp`, `interval_type`),
    INDEX `idx_open_time` (`open_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='K线表';

