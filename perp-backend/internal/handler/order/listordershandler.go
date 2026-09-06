package order

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"metanode/internal/logic/order"
	"metanode/internal/svc"
	"metanode/internal/types"
)

// 查询订单列表
func ListOrdersHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ListOrdersReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := order.NewListOrdersLogic(r.Context(), svcCtx)
		resp, err := l.ListOrders(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
