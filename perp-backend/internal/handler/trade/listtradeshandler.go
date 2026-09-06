package trade

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"metanode/internal/logic/trade"
	"metanode/internal/svc"
	"metanode/internal/types"
)

// 查询成交记录
func ListTradesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ListTradesReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := trade.NewListTradesLogic(r.Context(), svcCtx)
		resp, err := l.ListTrades(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
