package funding

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	fundinglogic "metanode/internal/logic/funding"
	"metanode/internal/svc"
	"metanode/internal/types"
)

func GetLatestFundingRateHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetLatestFundingRateReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		l := fundinglogic.NewGetLatestFundingRateLogic(r.Context(), svcCtx)
		resp, err := l.GetLatestFundingRate(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		httpx.OkJsonCtx(r.Context(), w, resp)
	}
}
