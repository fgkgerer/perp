package market

import (
	"net/http"

	marketlogic "metanode/internal/logic/market"
	"metanode/internal/svc"
	"metanode/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// GetOrderPreviewHandler 下单前滑点保护与深度模拟。
func GetOrderPreviewHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetOrderPreviewReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		l := marketlogic.NewGetOrderPreviewLogic(r.Context(), svcCtx)
		resp, err := l.GetOrderPreview(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, resp)
	}
}
