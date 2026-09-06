package listener

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"metanode/internal/chain"
	"metanode/internal/model"
	"metanode/internal/svc"

	"github.com/ethereum/go-ethereum/common"
	"github.com/zeromicro/go-zero/core/logx"
)

const redisTreasuryLastBlockKey = "metanode:treasury_deposit:last_block"

// TreasuryDepositWatcher 监听 USDC Transfer(to = Ethereum.UsdcTreasuryAddress)，写入 deposits 并累加 ledger_balances。
type TreasuryDepositWatcher struct {
	svc       *svc.ServiceContext
	stop      chan struct{}
	once      sync.Once
	tickCount int
}

func NewTreasuryDepositWatcher(svc *svc.ServiceContext) *TreasuryDepositWatcher {
	return &TreasuryDepositWatcher{
		svc:  svc,
		stop: make(chan struct{}),
	}
}

// Start 非阻塞启动。
func (w *TreasuryDepositWatcher) Start() {
	cfg := w.svc.Config.TreasuryDeposit
	if !cfg.Enabled {
		logx.Info("TreasuryDeposit watcher disabled")
		return
	}
	if w.svc.Chain == nil {
		logx.Error("TreasuryDeposit watcher skipped: chain client nil (check RpcUrl)")
		return
	}
	if strings.TrimSpace(w.svc.Config.Ethereum.UsdcAddress) == "" ||
		strings.TrimSpace(w.svc.Config.Ethereum.UsdcTreasuryAddress) == "" {
		logx.Error("TreasuryDeposit watcher skipped: UsdcAddress or UsdcTreasuryAddress empty")
		return
	}
	if w.svc.DepositModel == nil || w.svc.LedgerBalanceModel == nil {
		logx.Error("TreasuryDeposit watcher skipped: Supabase unavailable (DepositModel/LedgerBalanceModel nil)")
		return
	}

	logx.Infof(
		"TreasuryDeposit watcher starting: usdc=%s treasury=%s poll=%ds confirmations=%d lookback=%d",
		w.svc.Config.Ethereum.UsdcAddress,
		w.svc.Config.Ethereum.UsdcTreasuryAddress,
		w.pollSeconds(),
		w.confirmations(),
		w.lookback(),
	)
	go w.loop()
}

// Stop 停止轮询。
func (w *TreasuryDepositWatcher) Stop() {
	w.once.Do(func() {
		close(w.stop)
	})
}

func (w *TreasuryDepositWatcher) pollSeconds() int {
	n := w.svc.Config.TreasuryDeposit.PollIntervalSeconds
	if n <= 0 {
		return 12
	}
	return n
}

func (w *TreasuryDepositWatcher) confirmations() uint64 {
	c := w.svc.Config.TreasuryDeposit.Confirmations
	if c == 0 {
		return 1
	}
	return c
}

func (w *TreasuryDepositWatcher) lookback() uint64 {
	l := w.svc.Config.TreasuryDeposit.InitialLookbackBlocks
	if l == 0 {
		return 2000
	}
	return l
}

func (w *TreasuryDepositWatcher) loop() {
	interval := time.Duration(w.pollSeconds()) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	w.tickOnce(context.Background())

	for {
		select {
		case <-w.stop:
			return
		case <-ticker.C:
			w.tickOnce(context.Background())
		}
	}
}

func (w *TreasuryDepositWatcher) tickOnce(ctx context.Context) {
	w.tickCount++

	head, err := w.svc.Chain.RPC().BlockNumber(ctx)
	if err != nil {
		logx.Errorf("treasury watcher BlockNumber: %v", err)
		return
	}

	conf := w.confirmations()
	var safeHead uint64
	if head > conf {
		safeHead = head - conf
	} else {
		safeHead = 0
	}

	last, err := w.loadLastBlock(ctx)
	if err != nil {
		logx.Errorf("treasury watcher loadLastBlock: %v", err)
		return
	}

	fromBlock := last + 1
	if last == 0 {
		if safeHead > w.lookback() {
			fromBlock = safeHead - w.lookback()
		} else {
			fromBlock = 0
		}
	}

	if fromBlock > safeHead {
		if w.tickCount%30 == 0 {
			logx.Infof("treasury watcher idle: head=%d safe=%d last=%d", head, safeHead, last)
		}
		return
	}

	usdc := common.HexToAddress(strings.TrimSpace(w.svc.Config.Ethereum.UsdcAddress))
	treasury := common.HexToAddress(strings.TrimSpace(w.svc.Config.Ethereum.UsdcTreasuryAddress))

	transfers, err := w.svc.Chain.FilterUsdcTransfersTo(ctx, usdc, treasury, fromBlock, safeHead)
	if err != nil {
		logx.Errorf("treasury watcher FilterLogs blocks %d-%d: %v", fromBlock, safeHead, err)
		return
	}

	var applied int
	var applyErr bool
	for _, tr := range transfers {
		if err := w.applyTransfer(ctx, tr); err != nil {
			applyErr = true
			logx.Errorf("treasury watcher apply tx=%s log=%d: %v", tr.TxHash.Hex(), tr.LogIndex, err)
		} else {
			applied++
		}
	}
	if applyErr {
		return
	}

	w.saveLastBlock(ctx, safeHead)

	if applied > 0 || w.tickCount%20 == 0 {
		logx.Infof(
			"treasury watcher ok: scanned %d-%d, new_transfers=%d, last=%d",
			fromBlock, safeHead, applied, safeHead,
		)
	}
}

func (w *TreasuryDepositWatcher) loadLastBlock(ctx context.Context) (uint64, error) {
	s, err := w.svc.Redis.GetCtx(ctx, redisTreasuryLastBlockKey)
	if err != nil {
		logx.Infof("treasury watcher redis unavailable, fallback deposits max block: %v", err)
		return w.lastBlockFromDeposits(ctx)
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return w.lastBlockFromDeposits(ctx)
	}
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return w.lastBlockFromDeposits(ctx)
	}
	return n, nil
}

func (w *TreasuryDepositWatcher) lastBlockFromDeposits(ctx context.Context) (uint64, error) {
	if w.svc.DepositModel == nil {
		return 0, nil
	}
	n, err := w.svc.DepositModel.MaxBlockNumber(ctx)
	if err != nil {
		logx.Errorf("treasury watcher deposits max block: %v", err)
		return 0, nil
	}
	return n, nil
}

func (w *TreasuryDepositWatcher) saveLastBlock(ctx context.Context, block uint64) {
	if err := w.svc.Redis.SetCtx(ctx, redisTreasuryLastBlockKey, strconv.FormatUint(block, 10)); err != nil {
		logx.Errorf("treasury watcher redis set last_block=%d: %v (next tick may rescan)", block, err)
	}
}

func (w *TreasuryDepositWatcher) applyTransfer(ctx context.Context, tr chain.UsdcTreasuryTransfer) error {
	txHex := tr.TxHash.Hex()
	exist, err := w.svc.DepositModel.FindByTxHashAndLogIndex(ctx, txHex, tr.LogIndex)
	if err != nil {
		return err
	}
	if exist != nil {
		return nil
	}

	traderLower := strings.ToLower(tr.From.Hex())
	_, err = w.svc.DepositModel.Insert(ctx, &model.Deposit{
		TxHash:          txHex,
		LogIndex:        int64(tr.LogIndex),
		Trader:          traderLower,
		PrimaryAmount:   tr.Value.String(),
		SecondaryAmount: "0",
		BlockNumber:     int64(tr.BlockNumber),
		CreateTime:      time.Now(),
	})
	if err != nil {
		if model.IsDuplicateKeyErr(err) {
			return nil
		}
		return err
	}

	if err := w.svc.LedgerBalanceModel.AddPrimary(ctx, traderLower, tr.Value); err != nil {
		return err
	}
	logx.Infof("treasury deposit credited: trader=%s amount=%s tx=%s", traderLower, tr.Value.String(), txHex)
	return nil
}
