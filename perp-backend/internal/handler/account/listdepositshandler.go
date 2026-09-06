package account

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"metanode/internal/logic/account"
	"metanode/internal/svc"
	"metanode/internal/types"
)

// 查询存款记录
func ListDepositsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ListDepositsReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := account.NewListDepositsLogic(r.Context(), svcCtx)
		resp, err := l.ListDeposits(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, resp)
	}
}
