# MetaNode 后端 — 常驻服务（撮合 / 资金费 / WS）
FROM golang:1.24-alpine AS builder

WORKDIR /src
RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/metanode .

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app

COPY --from=builder /out/metanode /app/metanode
COPY etc/metanode.yaml /app/etc/metanode.yaml

ENV TZ=UTC
EXPOSE 28888

# 生产请挂载自己的配置：-v ./etc/metanode.prod.yaml:/app/etc/metanode.yaml:ro
CMD ["/app/metanode", "-f", "/app/etc/metanode.yaml"]
