package chain

// 最小 ABI 片段（与 src 中 IDealer / IPerpetual / Chainlink Aggregator 对齐），避免引入完整 abigen 生成代码。
// dealer：视图与 updateFundingRate；perpetual：trade / balanceOf；aggregator：latestRoundData。
const dealerABIStr = `[
  {"name":"getMarkPrice","type":"function","stateMutability":"view","inputs":[{"name":"perp","type":"address"}],"outputs":[{"name":"","type":"uint256"}]},
  {"name":"getCreditOf","type":"function","stateMutability":"view","inputs":[{"name":"trader","type":"address"}],"outputs":[
    {"name":"primaryCredit","type":"int256"},
    {"name":"secondaryCredit","type":"uint256"},
    {"name":"pendingPrimaryWithdraw","type":"uint256"},
    {"name":"pendingSecondaryWithdraw","type":"uint256"},
    {"name":"executionTimestamp","type":"uint256"}
  ]},
  {"name":"getTraderRisk","type":"function","stateMutability":"view","inputs":[{"name":"trader","type":"address"}],"outputs":[
    {"name":"netValue","type":"int256"},
    {"name":"exposure","type":"uint256"},
    {"name":"initialMargin","type":"uint256"},
    {"name":"maintenanceMargin","type":"uint256"}
  ]},
  {"name":"getPositions","type":"function","stateMutability":"view","inputs":[{"name":"trader","type":"address"}],"outputs":[{"name":"","type":"address[]"}]},
  {"name":"getFundingRate","type":"function","stateMutability":"view","inputs":[{"name":"perp","type":"address"}],"outputs":[{"name":"","type":"int256"}]},
  {"name":"getLiquidationPrice","type":"function","stateMutability":"view","inputs":[{"name":"trader","type":"address"},{"name":"perp","type":"address"}],"outputs":[{"name":"","type":"uint256"}]},
  {"name":"isSafe","type":"function","stateMutability":"view","inputs":[{"name":"trader","type":"address"}],"outputs":[{"name":"","type":"bool"}]},
  {"name":"updateFundingRate","type":"function","stateMutability":"nonpayable","inputs":[
    {"name":"perpList","type":"address[]"},
    {"name":"rateList","type":"int256[]"}
  ],"outputs":[]},
  {"name":"isOrderSenderValid","type":"function","stateMutability":"view","inputs":[{"name":"orderSender","type":"address"}],"outputs":[{"name":"","type":"bool"}]},
  {"name":"getRiskParams","type":"function","stateMutability":"view","inputs":[{"name":"perp","type":"address"}],"outputs":[{"components":[
    {"name":"initialMarginRatio","type":"uint256"},
    {"name":"liquidationThreshold","type":"uint256"},
    {"name":"liquidationPriceOff","type":"uint256"},
    {"name":"insuranceFeeRate","type":"uint256"},
    {"name":"markPriceSource","type":"address"},
    {"name":"name","type":"string"},
    {"name":"isRegistered","type":"bool"}
  ],"name":"","type":"tuple"}]}
]`

const perpetualABIStr = `[
  {"name":"trade","type":"function","stateMutability":"nonpayable","inputs":[{"name":"tradeData","type":"bytes"}],"outputs":[]},
  {"name":"balanceOf","type":"function","stateMutability":"view","inputs":[{"name":"trader","type":"address"}],"outputs":[
    {"name":"paper","type":"int256"},
    {"name":"credit","type":"int256"}
  ]}
]`

const aggregatorV3ABIStr = `[
  {"name":"latestRoundData","type":"function","stateMutability":"view","inputs":[],"outputs":[
    {"name":"roundId","type":"uint80"},
    {"name":"answer","type":"int256"},
    {"name":"startedAt","type":"uint256"},
    {"name":"updatedAt","type":"uint256"},
    {"name":"answeredInRound","type":"uint80"}
  ]}
]`
