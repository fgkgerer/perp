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
			nil,
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
		from := common.BytesToAddress(lg.Topics[1][12:])
		to := common.BytesToAddress(lg.Topics[2][12:])
		if to != treasury {
			continue
		}
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
