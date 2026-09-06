-- USDC 转收款地址 address_a 链下入账：deposits 增加 log_index，ledger_balances 账本。
ALTER TABLE deposits ADD COLUMN log_index INT UNSIGNED NOT NULL DEFAULT 0 AFTER tx_hash;
-- 若已有 uk_tx_hash 请先调整后替换；新库建议在 CREATE TABLE 时带 uk_tx_log。
ALTER TABLE deposits ADD UNIQUE KEY uk_tx_log (tx_hash, log_index);

CREATE TABLE IF NOT EXISTS ledger_balances (
  trader VARCHAR(42) NOT NULL COMMENT '小写 0x 地址',
  primary_balance VARCHAR(96) NOT NULL DEFAULT '0' COMMENT 'USDC 最小单位十进制字符串',
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (trader)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
