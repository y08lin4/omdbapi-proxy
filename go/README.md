# OMDb API 多 Key 代理管理器（Go）

这是 Go 版本，适合部署在 VPS、Docker 或自建服务器上。客户端请求格式保持 OMDb 官方风格，但 `apikey` 使用你自己发放的服务 key。服务端会从 `omdb_keys.txt` 中轮询选择 OMDb 官方 key，请求官方接口后原样返回。

## 1. 文件说明

```text
omdb_keys.txt       # OMDb 官方 key 池，一行一个，不要提交到 Git
client_keys.txt     # 你发给调用方的服务访问 key，一行一个，不要提交到 Git
.env                # 服务配置，不要提交到 Git
```

示例文件：

```text
omdb_keys.example.txt
client_keys.example.txt
.env.example
```

## 2. 配置步骤

进入 Go 目录：

```bash
cd go
```

复制示例配置：

```bash
cp .env.example .env
cp omdb_keys.example.txt omdb_keys.txt
cp client_keys.example.txt client_keys.txt
```

编辑 `omdb_keys.txt`，一行一个 OMDb 官方 key：

```text
omdb_key_1
omdb_key_2
omdb_key_3
```

编辑 `client_keys.txt`，一行一个你发给用户的访问 key：

```text
client_key_1
client_key_2
```

编辑 `.env`：

```env
LISTEN_ADDR=:8080
OMDB_KEYS_FILE=omdb_keys.txt
CLIENT_KEYS_FILE=client_keys.txt
ADMIN_KEY=change_me_admin_key
HTTP_TIMEOUT=10s
KEY_COOLDOWN=5m
CORS_ORIGIN=*
```

如果真实 `omdb_keys.txt` 放在项目根目录，可以设置：

```env
OMDB_KEYS_FILE=../omdb_keys.txt
```

## 3. 启动

直接运行：

```bash
go run .
```

构建二进制：

```bash
go build -o omdb-api-manager .
```

Docker：

```bash
docker compose up -d --build
```

访问文档：

```text
http://localhost:8080/docs
```

## 4. 请求示例

按标题查询：

```text
http://localhost:8080/?apikey=CLIENT_KEY&t=Inception&plot=full
```

按 IMDb ID 查询：

```text
http://localhost:8080/?apikey=CLIENT_KEY&i=tt1375666
```

搜索：

```text
http://localhost:8080/?apikey=CLIENT_KEY&s=Batman&page=2
```

剧集季 / 集：

```text
http://localhost:8080/?apikey=CLIENT_KEY&t=Game%20of%20Thrones&Season=1
http://localhost:8080/?apikey=CLIENT_KEY&t=Game%20of%20Thrones&Season=1&Episode=1
```

Poster API：

```text
http://localhost:8080/poster?apikey=CLIENT_KEY&i=tt1375666
```

## 5. 管理接口

管理接口需要 `.env` 中的 `ADMIN_KEY`。

### 查看统计

```text
GET /admin/stats?admin_key=ADMIN_KEY
```

示例：

```bash
curl "http://localhost:8080/admin/stats?admin_key=ADMIN_KEY"
```

返回内容包括：

- 客户端 key 数量。
- OMDb key 总数。
- 可用 OMDb key 数量。
- 每个 OMDb key 的脱敏值、请求次数、成功次数、失败次数、冷却状态。
- 今日请求数和总请求数。

### 重载 key 文件

```text
POST /admin/reload?admin_key=ADMIN_KEY
```

示例：

```bash
curl -X POST "http://localhost:8080/admin/reload?admin_key=ADMIN_KEY"
```

作用：重新读取 `omdb_keys.txt` 和 `client_keys.txt`。当你新增或删除 key 后，可以调用这个接口，不需要重启服务。

## 6. 公开状态接口

健康检查：

```text
GET /health
```

数据看板接口：

```text
GET /metrics
```

返回：

```json
{
  "requests": {
    "total": 10,
    "today": 3,
    "day": "2026-04-30"
  }
}
```

## 7. 负载与容错

- OMDb key 默认轮询使用。
- 如果某个 OMDb key 返回额度耗尽、无效 key、429、5xx 或超时，会进入冷却，自动尝试下一个 key。
- 普通业务错误不会切 key，例如 `Movie not found!` 会原样返回给客户端。
- 客户端 key 不限流。

## 8. 公网部署建议

建议把 Go 服务放在 Caddy 或 Nginx 后面做 HTTPS。

Caddy 示例：

```caddyfile
api.example.com {
    reverse_proxy 127.0.0.1:8080
}
```

## 9. 注意

当前请求统计是内存统计，进程重启后会清零。如果需要持久统计，可以后续接入数据库。
