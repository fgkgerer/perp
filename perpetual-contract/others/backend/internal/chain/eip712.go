package chain

// 与 src/libraries/Trading.sol、Types.sol、MetaNodeStorage 中的 EIP-712 定义对齐，用于链下构造订单哈希与验签。

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var orderTypeHash = crypto.Keccak256Hash([]byte("Order(address perp,address signer,int128 paperAmount,int128 creditAmount,bytes32 info)"))

// padUint256 将 chainId 等填充为 32 字节大端，与 abi.encode(uint256) 一致。
func padUint256(n *big.Int) []byte {
	var b [32]byte
	if n != nil {
		n.FillBytes(b[:])
	}
	return b[:]
}

// HashDomainSeparator 计算 Dealer 合约中的 immutable domainSeparator。
func HashDomainSeparator(chainID *big.Int, dealer common.Address) common.Hash {
	typeHash := crypto.Keccak256Hash([]byte("EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"))
	nameHash := crypto.Keccak256Hash([]byte("MetaNode"))
	versionHash := crypto.Keccak256Hash([]byte("1"))
	payload := append(append(append(append(append([]byte{}, typeHash[:]...), nameHash[:]...), versionHash[:]...), padUint256(chainID)...), common.LeftPadBytes(dealer.Bytes(), 32)...)
	return crypto.Keccak256Hash(payload)
}

// PackOrderInfo 组装 Types.Order.info：高→低 64bit 依次为 makerFee、takerFee、expiration、nonce（与合约解析一致）。
func PackOrderInfo(makerFeeBps, takerFeeBps int64, expiration, nonce uint64) [32]byte {
	n := new(big.Int).SetUint64(uint64(makerFeeBps))
	n.Lsh(n, 192)
	t := new(big.Int).SetUint64(uint64(takerFeeBps))
	t.Lsh(t, 128)
	n.Or(n, t)
	e := new(big.Int).SetUint64(expiration)
	e.Lsh(e, 64)
	n.Or(n, e)
	nu := new(big.Int).SetUint64(nonce)
	n.Or(n, nu)
	var out [32]byte
	n.FillBytes(out[:])
	return out
}

// HashOrderStruct 计算 EIP-712 struct hash（不含 domain 包装）。
func HashOrderStruct(perp, signer common.Address, paperAmount, creditAmount *big.Int, info [32]byte) common.Hash {
	t, _ := abi.NewType("bytes32", "", nil)
	tAddr, _ := abi.NewType("address", "", nil)
	tI256, _ := abi.NewType("int256", "", nil)
	tB32, _ := abi.NewType("bytes32", "", nil)
	args := abi.Arguments{
		{Type: t}, {Type: tAddr}, {Type: tAddr}, {Type: tI256}, {Type: tI256}, {Type: tB32},
	}
	// int128 在内存中符号扩展为 32 字节，与 abi.encode(int256) 一致
	pap := new(big.Int).Set(paperAmount)
	cre := new(big.Int).Set(creditAmount)
	packed, err := args.Pack(orderTypeHash, perp, signer, pap, cre, info)
	if err != nil {
		panic(err)
	}
	return crypto.Keccak256Hash(packed)
}

// SignableOrderHash 用户应签名的 32 字节 digest（EIP-712 v4）。
func SignableOrderHash(chainID *big.Int, dealer, perp, signer common.Address, paperAmount, creditAmount *big.Int, info [32]byte) common.Hash {
	domain := HashDomainSeparator(chainID, dealer)
	structHash := HashOrderStruct(perp, signer, paperAmount, creditAmount, info)
	return crypto.Keccak256Hash(
		append(append([]byte{0x19, 0x01}, domain[:]...), structHash[:]...),
	)
}

// ParseIntString 解析十进制整数字符串（可为负）。
func ParseIntString(s string) (*big.Int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("empty int")
	}
	z := new(big.Int)
	_, ok := z.SetString(s, 10)
	if !ok {
		return nil, fmt.Errorf("invalid int: %s", s)
	}
	return z, nil
}

// ParseFeeInt64 从配置字符串解析为 int64（用于打包 info，通常为带 1e18 精度的费率）。
func ParseFeeInt64(s string) (int64, error) {
	s = strings.TrimSpace(s)
	z := new(big.Int)
	if _, ok := z.SetString(s, 10); !ok {
		return 0, fmt.Errorf("fee: %s", s)
	}
	if !z.IsInt64() {
		return 0, fmt.Errorf("fee too large for int64")
	}
	return z.Int64(), nil
}
