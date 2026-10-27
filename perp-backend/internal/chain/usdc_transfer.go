package chain

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// ERC20 Transfer(address indexed from, address indexed to, uint256 value)
var erc20TransferSig = crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))

// UsdcTreasuryTransfer 一笔转入收款地址的 USDC Transfer 日志。
type UsdcTreasuryTransfer struct {
	TxHash      common.Hash
	BlockNumber uint64
	LogIndex    uint
	From        common.Address
	To          common.Address
	Value       *big.Int
}

// FilterUsdcTransfersTo 筛选 USDC 合约上 to == treasury 的 Transfer（from 为用户钱包）。
func (c *Client) FilterUsdcTransfersTo(ctx context.Context, usdc, treasury common.Address, fromBlock, toBlock uint64) ([]UsdcTreasuryTransfer, error) {
	topicTo := common.BytesToHash(common.LeftPadBytes(treasury.Bytes(), 32))
	q := ethereum.FilterQuery{
		FromBlock: new(big.Int).SetUint64(fromBlock),
		ToBlock:   new(big.Int).SetUint64(toBlock),
		Addresses: []common.Address{usdc},
		Topics: [][]common.Hash{
			{erc20TransferSig},
			nil, //nil=不限制from的用户地址
			{topicTo},
		},
	}
	logs, err := c.eth.FilterLogs(ctx, q)
	if err != nil {
		return nil, err
	}
	out := make([]UsdcTreasuryTransfer, 0, len(logs))
	for _, lg := range logs {
		if len(lg.Topics) != 3 || len(lg.Data) < 32 {
			continue
		}
		/** lg.Topics[1][12:] 为什么是 [12:]？
		因为 以太坊地址是 20 字节，而 indexed 参数在日志 topic 里会被存成 32 字节。所以一个地址放进 topic 后，
		会变成：[ 12字节补0 ][ 20字节地址 ]
		*/
		from := common.BytesToAddress(lg.Topics[1][12:])
		to := common.BytesToAddress(lg.Topics[2][12:])
		if to != treasury {
			continue
		}
		/** value 不是 indexed，所以会放进日志的 Data 字段。
				value 是 uint256，占 32 字节，所以这里用：lg.Data[:32]
				把它转成 big.Int，也就是转账金额

		| 代码 | 含义 |
		|---|---|
		| `lg.Topics[1][12:]` | 从 32 字节 topic 中取出右边 20 字节，得到 `from` 地址 |
		| `lg.Data[:32]` | 取日志 `Data` 的前 32 字节，得到 `value` 金额 |

		*/
		val := new(big.Int).SetBytes(lg.Data[:32])
		out = append(out, UsdcTreasuryTransfer{
			TxHash:      lg.TxHash,
			BlockNumber: lg.BlockNumber,
			LogIndex:    lg.Index,
			From:        from,
			To:          to,
			Value:       val,
		})
	}
	return out, nil
}
