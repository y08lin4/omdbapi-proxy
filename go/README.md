# OMDb API 多 Key 代理管理器（Go）

这是一个公网可部署的 OMDb API 代理服务：客户端请求格式保持 OMDb 官方风格，但 `apikey` 使用你自己发放的服务 key。服务端会从 `omdb_keys.txt` 中轮询选择 OMDb 官方 key，请求官方接口后原样返回。

## 已支持的 API

- `GET /`：OMDb 数据 API，兼容官方 query 参数。
- `GET /api`：数据 API 别名。
- `GET /poster`：OMDb poster API 代理。
- `GET /docs`：静态请求文档页面。
- `GET /health`：健康检查。
- `GET /admin/stats`：查看 key 池状态，需要 admin key。
- `POST /admin/reload`：重载 `omdb_keys.txt` 和 `client_keys.txt`，需要 admin key。

数据 API 透传所有官方参数，例如：`i`、`t`、`s`、`type`、`y`、`plot`、`r`、`callback`、`v`、`page`、`Season`、`Episode` 等。

## Key 文件

### `omdb_keys.txt`

OMDb 官方 key 池，一行一个：

```txt
omdb_key_1
omdb_key_2
omdb_key_3
```

### `client_keys.txt`

你发给调用方的服务访问 key，一行一个：

```txt
client_key_1
client_key_2
```

没有客户端 key 或 key 错误时，代理 API 会直接返回 `401`，不会请求 OMDb。

## 配置

复制示例：

```powershell
Copy-Item .env.example .env
Copy-Item omdb_keys.example.txt omdb_keys.txt
Copy-Item client_keys.example.txt client_keys.txt
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

## 启动

```powershell
go run .
```

构建：

```powershell
go build -o omdb-api-manager.exe .
```

访问文档：

```text
http://localhost:8080/docs
```

## 请求示例

### 按标题查询

```text
GET http://localhost:8080/?apikey=YOUR_CLIENT_KEY&t=Inception&plot=full
```

### 按 IMDb ID 查询

```text
GET http://localhost:8080/?apikey=YOUR_CLIENT_KEY&i=tt1375666
```

### 搜索

```text
GET http://localhost:8080/?apikey=YOUR_CLIENT_KEY&s=Batman&page=2
```

### 剧集季/集

```text
GET http://localhost:8080/?apikey=YOUR_CLIENT_KEY&t=Game%20of%20Thrones&Season=1
GET http://localhost:8080/?apikey=YOUR_CLIENT_KEY&t=Game%20of%20Thrones&Season=1&Episode=1
```

### XML / JSONP

```text
GET http://localhost:8080/?apikey=YOUR_CLIENT_KEY&t=Inception&r=xml
GET http://localhost:8080/?apikey=YOUR_CLIENT_KEY&t=Inception&callback=myCallback
```

### Poster API

```text
GET http://localhost:8080/poster?apikey=YOUR_CLIENT_KEY&i=tt1375666
```

### Header 传 key

也可以把服务 key 放在请求头里：

```text
GET /?t=Inception HTTP/1.1
X-API-Key: YOUR_CLIENT_KEY
```

或：

```text
Authorization: Bearer YOUR_CLIENT_KEY
```

## 管理接口

### 查看状态

```text
GET http://localhost:8080/admin/stats?admin_key=ADMIN_KEY
```

### 重载 key 文件

```text
POST http://localhost:8080/admin/reload?admin_key=ADMIN_KEY
```

## Docker 部署

```powershell
docker build -t omdb-api-manager .
docker run -d --name omdb-api-manager `
  -p 8080:8080 `
  -v ${PWD}/omdb_keys.txt:/app/omdb_keys.txt:ro `
  -v ${PWD}/client_keys.txt:/app/client_keys.txt:ro `
  --env-file .env `
  omdb-api-manager
```

公网建议放在 Caddy/Nginx 后面做 HTTPS。

## 负载与容错

- OMDb key 默认轮询使用。
- 如果某个 OMDb key 返回额度耗尽、无效 key、429、5xx 或超时，会进入冷却，自动尝试下一个 key。
- 普通业务错误不会切 key，例如 `Movie not found!` 会原样返回给客户端。
- 客户端 key 不限流。
