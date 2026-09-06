package market

import (
	"context"
	"math/big"
	"strings"

	"metanode/internal/engine"
	"metanode/internal/svc"
	"metanode/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetOrderPreviewLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetOrderPreviewLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetOrderPreviewLogic {
	return &GetOrderPreviewLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetOrderPreviewLogic) GetOrderPreview(req *types.GetOrderPreviewReq) (*types.GetOrderPreviewResp, error) {
	perp := strings.TrimSpace(req.Perp)
	side := strings.TrimSpace(req.Side)
	if perp == "" {
		return &types.GetOrderPreviewResp{Code: 400, Message: "perp is required"}, nil
	}
	if side != "long" && side != "short" {
		return &types.GetOrderPreviewResp{Code: 400, Message: "side must be long or short"}, nil
	}

	sizePaper, err := ParsePaperHuman(req.Size)
	if err != nil {
		return &types.GetOrderPreviewResp{Code: 400, Message: "invalid size"}, nil
	}

	slippageBps := NormalizeSlippageBps(req.SlippageBps)
	takerIsBuy := sideIsLong(side)

	q := ResolveOpenQuote(l.ctx, l.svcCtx, perp, side)
	refRaw, ok := new(big.Int).SetString(strings.TrimSpace(q.PriceRaw), 10)
	if !ok || refRaw.Sign() <= 0 {
		return &types.GetOrderPreviewResp{Code: 404, Message: "no reference price available"}, nil
	}

	limitRaw := LimitPriceWithSlippage(refRaw, takerIsBuy, slippageBps)
	skipSigner := strings.TrimSpace(req.Signer)

	var sim engine.FillSimulation
	if me, ok := l.svcCtx.OrderBook.(*engine.MatchEngine); ok {
		sim = me.SimulateTakerFill(perp, takerIsBuy, sizePaper, limitRaw, skipSigner)
	} else {
		sim = engine.FillSimulation{UnfilledPaper: new(big.Int).Set(sizePaper)}
	}

	fills := make([]types.OrderPreviewFill, 0, len(sim.Levels))
	for _, lv := range sim.Levels {
		amt, _ := new(big.Int).SetString(lv.AmountPaper, 10)
		fills = append(fills, types.OrderPreviewFill{
			PriceUsd:  FormatMarkDisplay(lv.PriceRaw),
			PriceRaw:  lv.PriceRaw,
			Amount:    FormatPaperHuman(amt),
			AmountRaw: lv.AmountPaper,
		})
	}

	filled := sim.FilledPaper
	if filled == nil {
		filled = big.NewInt(0)
	}
	unfilled := sim.UnfilledPaper
	if unfilled == nil {
		unfilled = big.NewInt(0)
	}

	preview := types.OrderPreview{
		Perp:              perp,
		Side:              side,
		SlippageBps:       slippageBps,
		ReferencePriceUsd: q.PriceUsd,
		ReferencePriceRaw: refRaw.String(),
		ReferenceSource:   q.Source,
		LimitPriceUsd:     FormatMarkDisplay(limitRaw.String()),
		LimitPriceRaw:     limitRaw.String(),
		RequestedSize:     strings.TrimSpace(req.Size),
		FilledSize:        FormatPaperHuman(filled),
		UnfilledSize:      FormatPaperHuman(unfilled),
		FullyFillable:     unfilled.Sign() == 0 && filled.Sign() > 0,
		Fills:             fills,
	}
	if sim.AvgPriceRaw != nil && sim.AvgPriceRaw.Sign() > 0 {
		preview.AvgFillPriceUsd = FormatMarkDisplay(sim.AvgPriceRaw.String())
	}
	if sim.WorstPriceRaw != nil && sim.WorstPriceRaw.Sign() > 0 {
		preview.WorstFillPriceUsd = FormatMarkDisplay(sim.WorstPriceRaw.String())
	}

	return &types.GetOrderPreviewResp{
		Code:    0,
		Message: "ok",
		Preview: preview,
	}, nil
}
