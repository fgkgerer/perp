package market

import (
	"net/http"
	"time"

	marketlogic "metanode/internal/logic/market"
	"metanode/internal/svc"

	"github.com/gorilla/websocket"
	"github.com/zeromicro/go-zero/rest"
)

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// 默认放开跨域便于本地联调；上线请改为校验 Origin。
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// RegisterTickerWebSocket 注册 GET /api/v1/ws/ticker：WebSocket 订阅市场价格（链上标记价）。
func RegisterTickerWebSocket(server *rest.Server, serverCtx *svc.ServiceContext) {
	server.AddRoute(rest.Route{
		Method: http.MethodGet,
		Path:   "/api/v1/ws/ticker",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			c, err := wsUpgrader.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			defer c.Close()
			_ = c.SetReadDeadline(time.Now().Add(60 * time.Second))
			for {
				// 定时从链上刷新各市场 MarkPrice（无鉴权；生产应校验 Origin 或按用户隔离）。
				tickers := marketlogic.ChainTickers(r.Context(), serverCtx)
				if err := c.WriteJSON(map[string]interface{}{
					"tickers": tickers,
					"time":    time.Now().Unix(),
				}); err != nil {
					return
				}
				time.Sleep(2 * time.Second)
			}
		},
	})
}
