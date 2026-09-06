package position

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"metanode/internal/logic/position"
	"metanode/internal/svc"
	"metanode/internal/types"
)

// 查询用户仓位
func GetPositionsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetPositionsReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := position.NewGetPositionsLogic(r.Context(), svcCtx)
		resp, err := l.GetPositions(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
