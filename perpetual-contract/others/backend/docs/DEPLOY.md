# MetaNode Backend 部署说明

## 结论：不建议部署到 Vercel

`others/backend` 是 **Go 常驻进程**（go-zero），不是 Next.js / Serverless API。启动后会同时跑：

| 组件 | 说明 |
|------|------|
| HTTP API | `:28888`，REST + CORS |
| WebSocket | 行情 Ticker |
| MatchEngine | 内存撮合，定时落库/上链 |
| FundingRateKeeper | 定时资金费结算 |
| Liquidator | 清算轮询 |
| TreasuryDepositWatcher | 扫链充值 |
| SpotIndex / Coinbase | 长连接 WS 拉指数价 |

还依赖 **Redis**（`redis.MustNewRedis`，启动即连）、**Postgres（Supabase）**、**Sepolia RPC**、**私钥**（撮合/资金费）。

Vercel 适合无状态 Serverless / 前端静态站，**不支持**这种多后台任务 + 长连接 + 单进程撮合引擎。硬上 Vercel 只能拆掉大部分能力，**不等于**当前仓库的完整后端。

### 推荐分工

| 部分 | 平台 |
|------|------|
| 前端 `others/fe` | **Vercel**（合适） |
| 后端 `others/backend` | **Railway / Render / Fly.io / 云主机 Docker** |

前端环境变量：

```bash
NEXT_PUBLIC_METANODE_API_URL=https://你的后端域名
NEXT_PUBLIC_SUPABASE_URL=...
NEXT_PUBLIC_SUPABASE_ANON_KEY=...
```

---

## 方式一：Docker（通用，推荐）

### 1. 构建镜像

```bash
cd others/backend
docker build -t metanode-backend .
```

### 2. 准备配置

复制并编辑配置（**勿把私钥提交 Git**）：

```bash
cp etc/metanode.yaml etc/metanode.prod.yaml
# 编辑：Redis、Supabase.DataSource、Ethereum.*、Auth.JwtSecret
```

生产环境 Redis 示例（Docker Compose 内网）：

```yaml
Redis:
  Host: redis:6379
  Type: node
  Pass: ""
```

### 3. 运行

```bash
docker run --rm -p 28888:28888 \
  -v "$(pwd)/etc/metanode.prod.yaml:/app/etc/metanode.yaml:ro" \
  metanode-backend
```

或使用仓库内 `docker-compose.yml`（含 Redis）：

```bash
cd others/backend
docker compose up -d --build
```

健康检查：`curl http://127.0.0.1:28888/api/v1/markets`

---

## 方式二：Railway（上手快）

1. [Railway](https://railway.app) 新建 Project → **Deploy from GitHub**
2. Root Directory 设为 `others/backend`（或 Dockerfile 路径指向该目录）
3. 添加 **Redis** 插件，把 `REDIS_URL` 填到环境变量后，在 `metanode.prod.yaml` 里写 `Redis.Host`（或后续支持 env 覆盖）
4. 在 Variables 配置敏感项（不要写进 yaml 再提交）：
   - 建议把 `etc/metanode.prod.yaml` 放 Volume，或用 Railway 的 Config 挂载
5. 暴露端口 **28888**，生成公网域名
6. 前端 Vercel 的 `NEXT_PUBLIC_METANODE_API_URL` 指向该域名

Railway 按容器常驻运行，**可以**跑撮合与定时任务（与 Vercel 不同）。

---

## 方式三：Render（Web Service）

1. New → **Web Service** → 连接仓库
2. Root Directory: `others/backend`
3. Environment: **Docker**
4. 实例类型选 **Starter** 或以上（需要常驻）
5. 添加 **Redis**（Render Key Value）并改 yaml 中 `Redis.Host`
6. 环境变量 / Secret File 挂载 `metanode.prod.yaml`
7. Health Check Path: `/api/v1/markets`

---

## 方式四：云主机（systemd）

```bash
cd others/backend
go build -o bin/metanode .
sudo cp etc/metanode.prod.yaml /etc/metanode/metanode.yaml
```

`/etc/systemd/system/metanode.service`：

```ini
[Unit]
Description=MetaNode Backend
After=network.target

[Service]
Type=simple
WorkingDirectory=/opt/metanode/backend
ExecStart=/opt/metanode/backend/bin/metanode -f /etc/metanode/metanode.yaml
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

前置：本机或内网 **Redis**、可访问 **Supabase Postgres**、出站 **HTTPS RPC**。

---

## 生产配置检查清单

- [ ] `Ethereum.RpcUrl`：稳定 Sepolia RPC（建议 Alchemy/Infura，勿用公开节点限流）
- [ ] `Ethereum.PrivateKey`：链上 `validOrderSender` / `fundingRateKeeper` 已配置
- [ ] `Supabase.DataSource`：`sslmode=require`，IP 白名单或 pooler
- [ ] `Redis`：可连（本地 / 云 Redis）
- [ ] `Auth.JwtSecret`：强随机，勿用示例值
- [ ] `TreasuryDeposit.Enabled`：按需开关扫链
- [ ] 防火墙只暴露 API 端口；私钥仅服务端可见
- [ ] 前端 CORS：当前代码为 `Allow-Origin: *`，生产可收紧

---

## 若坚持要用 Vercel 托管「后端」

仅能做 **极小规模实验**，需另开项目重写为 Vercel Serverless Functions（Go/Node），且：

- 不能跑 MatchEngine / 资金费 Keeper / 清算 / 扫链 / 指数 WS
- 订单撮合、WS 行情需迁到其他服务
- 函数超时（10s～60s）限制链上交易

**不等于** 迁移本仓库，工作量接近重写。务实方案仍是：**前端 Vercel + 后端 Docker 云**。

---

## 前端部署到 Vercel（简要）

```bash
cd others/fe
# 在 Vercel 项目 Settings → Environment Variables 配置 API / Supabase
vercel deploy
```

`vercel.json` 可选（根目录在 `others/fe` 时）：

```json
{
  "framework": "nextjs",
  "buildCommand": "npm run build",
  "outputDirectory": ".next"
}
```

Vercel 项目 **Root Directory** 设为 `others/fe`。
