package chain

// trade 与订单编码：对应 MetaNodeExternal.approveTrade 对 tradeData 的 abi.decode(Order[],bytes[],uint256[])。

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"metanode/internal/model"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// SolidityOrder 对应 Solidity Types.Order，用于与链上 abi.encode 一致。
type SolidityOrder struct {
	Perp         common.Address
	Signer       common.Address
	PaperAmount  *big.Int
	CreditAmount *big.Int
	Info         [32]byte
}

var tradeTupleType, _ = abi.NewType("tuple[]", "", []abi.ArgumentMarshaling{
	{Name: "perp", Type: "address"},
	{Name: "signer", Type: "address"},
	{Name: "paperAmount", Type: "int128"},
	{Name: "creditAmount", Type: "int128"},
	{Name: "info", Type: "bytes32"},
})

// EncodeTradeData 生成 perpetual.trade(tradeData) 所需的 abi.encode(orderList, signatureList, matchPaperAmount)。
func EncodeTradeData(orders []SolidityOrder, sigs [][]byte, matchPaper []*big.Int) ([]byte, error) {
	if len(orders) != len(sigs) || len(orders) != len(matchPaper) {
		return nil, fmt.Errorf("trade data length mismatch")
	}
	bt, _ := abi.NewType("bytes[]", "", nil)
	ut, _ := abi.NewType("uint256[]", "", nil)
	args := abi.Arguments{
		{Type: tradeTupleType},
		{Type: bt},
		{Type: ut},
	}
	type row struct {
		Perp         common.Address
		Signer       common.Address
		PaperAmount  *big.Int
		CreditAmount *big.Int
		Info         [32]byte
	}
	goOrders := make([]row, len(orders))
	for i := range orders {
		goOrders[i].Perp = orders[i].Perp
		goOrders[i].Signer = orders[i].Signer
		goOrders[i].PaperAmount = orders[i].PaperAmount
		goOrders[i].CreditAmount = orders[i].CreditAmount
		goOrders[i].Info = orders[i].Info
	}
	return args.Pack(goOrders, sigs, matchPaper)
}

// OrderFromModel 将 REST/DB 订单转为链上结构（info 字段按合约打包规则）。
func OrderFromModel(o *model.Order) (SolidityOrder, error) {
	perp := common.HexToAddress(o.Perp)
	signer := common.HexToAddress(o.Signer)
	paper, err := ParseIntString(o.PaperAmount)
	if err != nil {
		return SolidityOrder{}, err
	}
	credit, err := ParseIntString(o.CreditAmount)
	if err != nil {
		return SolidityOrder{}, err
	}
	mf, err := ParseFeeInt64(o.MakerFeeRate)
	if err != nil {
		return SolidityOrder{}, err
	}
	tf, err := ParseFeeInt64(o.TakerFeeRate)
	if err != nil {
		return SolidityOrder{}, err
	}
	info := PackOrderInfo(mf, tf, uint64(o.Expiration), uint64(o.Nonce))
	return SolidityOrder{
		Perp: perp, Signer: signer, PaperAmount: paper, CreditAmount: credit, Info: info,
	}, nil
}

// DealerChain Dealer 地址与链 ID（EIP-712）。
type DealerChain struct {
	Dealer  common.Address
	ChainID *big.Int
}

// OrderDigestWith 计算用户应对其签名的 EIP-712 digest，并用作 orderId。
func OrderDigestWith(dc DealerChain, o *model.Order) (common.Hash, error) {
	so, err := OrderFromModel(o)
	if err != nil {
		return common.Hash{}, err
	}
	return SignableOrderHash(dc.ChainID, dc.Dealer, so.Perp, so.Signer, so.PaperAmount, so.CreditAmount, so.Info), nil
}

// VerifyEOASignature 验证订单由 signer EOA 签名（不支持合约钱包的 EIP-1271）。
func VerifyEOASignature(dc DealerChain, o *model.Order) error {
	digest, err := OrderDigestWith(dc, o)
	if err != nil {
		return err
	}
	sig, err := DecodeSignature(o.Signature)
	if err != nil {
		return err
	}
	pub, err := crypto.SigToPub(digest.Bytes(), sig)
	if err != nil {
		return err
	}
	rec := crypto.PubkeyToAddress(*pub)
	if !strings.EqualFold(rec.Hex(), o.Signer) {
		return fmt.Errorf("signer mismatch: recovered=%s expect=%s", rec.Hex(), o.Signer)
	}
	return nil
}

// DecodeSignature 解析 hex 签名，归一化为 go-ethereum 要求的 [R||S||V]（V=0/1）。
func DecodeSignature(hexSig string) ([]byte, error) {
	h := strings.TrimSpace(hexSig)
	h = strings.TrimPrefix(h, "0x")
	b, err := hex.DecodeString(h)
	if err != nil {
		return nil, err
	}
	if len(b) == 64 {
		b = append(b, 0)
	}
	if len(b) != 65 {
		return nil, fmt.Errorf("bad sig length %d", len(b))
	}
	switch b[64] {
	case 27, 28:
		b[64] -= 27
	case 0, 1:
		// MetaMask / viem 常见格式
	case 2, 3:
		// EIP-155 recovery id，取 parity
		b[64] -= 2
	default:
		return nil, fmt.Errorf("bad recovery id %d", b[64])
	}
	return b, nil
}

// SignatureForChain 将签名转为链上 ECDSA.tryRecover 接受的格式（V=27/28）。
func SignatureForChain(hexSig string) ([]byte, error) {
	sig, err := DecodeSignature(hexSig)
	if err != nil {
		return nil, err
	}
	if sig[64] < 27 {
		sig[64] += 27
	}
	return sig, nil
}

// BuildMatchTradeData 将一笔撮合（taker + maker）编码为 tradeData。orderList[0] 必须为 taker。
func BuildMatchTradeData(taker, maker *model.Order, matchPaper *big.Int) ([]byte, error) {
	if taker.Perp != maker.Perp {
		return nil, fmt.Errorf("perp mismatch")
	}
	o0, err := OrderFromModel(taker)
	if err != nil {
		return nil, fmt.Errorf("taker: %w", err)
	}
	o1, err := OrderFromModel(maker)
	if err != nil {
		return nil, fmt.Errorf("maker: %w", err)
	}
	sigT, err := SignatureForChain(taker.Signature)
	if err != nil {
		return nil, err
	}
	sigM, err := SignatureForChain(maker.Signature)
	if err != nil {
		return nil, err
	}
	orders := []SolidityOrder{o0, o1}
	sigs := [][]byte{sigT, sigM}
	amt := new(big.Int).Abs(matchPaper)
	return EncodeTradeData(orders, sigs, []*big.Int{amt, amt})
}
