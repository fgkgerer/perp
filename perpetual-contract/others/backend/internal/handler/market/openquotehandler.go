package market

import (
	"net/http"

	"metanode/internal/logic/market"
	"metanode/internal/svc"
	"metanode/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// GetOpenQuoteHandler 开仓参考价：做多=卖一，做空=买一，供前端自动填限价。
func GetOpenQuoteHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetOpenQuoteReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		l := market.NewGetOpenQuoteLogic(r.Context(), svcCtx)
		resp, err := l.GetOpenQuote(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, resp)
	}
}
