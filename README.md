# OMDb API 管理器：Go 版 + Cloudflare Worker 版

[![Deploy to Cloudflare](https://deploy.workers.cloudflare.com/button)](https://deploy.workers.cloudflare.com/?url=https://github.com/y08lin4/omdbapi-proxy)

本项目是一个 OMDb API 代理管理器，提供两个部署版本：

```text
go/   # Go 公网部署版：适合 VPS、Docker、自建服务器；从 txt 文件读取 key
cf/   # Cloudflare Worker 版：适合边缘部署；从 Worker 环境变量 / Secrets 读取 key
```

客户端请求格式保持 OMDb 官方风格：

```text
GET /?apikey=YOUR_CLIENT_KEY&t=Inception
GET /?apikey=YOUR_CLIENT_KEY&s=Batman&page=2
GET /?apikey=YOUR_CLIENT_KEY&i=tt1375666
GET /poster?apikey=YOUR_CLIENT_KEY&i=tt1375666
```

这里的 `apikey` 是你发给调用方的服务 key，不是 OMDb 官方 key。服务端或 Worker 会自动替换成内部 OMDb 官方 key。

## 1. Key 的角色说明

本项目有三类 key：

| 名称 | 用途 | 给谁用 | 是否应该公开 |
| --- | --- | --- | --- |
| `CLIENT_KEYS` | 你的代理服务访问 key | 发给你的 API 调用方 | 不建议公开 |
| `OMDB_KEYS` | OMDb 官方 key 池 | 只给代理服务内部使用 | 绝对不要公开 |
| `ADMIN_KEY` | 管理接口 key | 只给管理员使用 | 绝对不要公开 |

请求代理 API 时，用户传的是 `CLIENT_KEYS` 里的 key：

```text
https://你的域名/?apikey=CLIENT_KEY&t=Inception
```

代理服务内部会把它替换成 `OMDB_KEYS` 里的某个 OMDb 官方 key。

## 2. 管理接口是干什么的？

管理接口用于查看服务内部状态和重载 key，不是给普通用户调用的。

### `/admin/stats`

查看当前状态，包括：

- 有多少客户端 key。
- OMDb key 池总数。
- 当前可用 OMDb key 数。
- 每个 OMDb key 的请求次数、成功次数、失败次数、冷却状态。
- 今日请求数、总请求数。

请求方式：

```text
GET /admin/stats?admin_key=YOUR_ADMIN_KEY
```

也可以用请求头：

```text
X-Admin-Key: YOUR_ADMIN_KEY
```

示例：

```bash
curl "https://你的域名/admin/stats?admin_key=YOUR_ADMIN_KEY"
```

### `/admin/reload`

重新加载 key。

- Go 版：重新读取 `omdb_keys.txt` 和 `client_keys.txt`。
- Cloudflare Worker 版：重新解析当前环境变量 / Secrets 中的 `OMDB_KEYS` 和 `CLIENT_KEYS`，并刷新内存状态。

请求方式：

```text
POST /admin/reload?admin_key=YOUR_ADMIN_KEY
```

示例：

```bash
curl -X POST "https://你的域名/admin/reload?admin_key=YOUR_ADMIN_KEY"
```

什么时候用？

- 新增或删除客户端 key 后。
- 新增或删除 OMDb 官方 key 后。
- 想让服务立刻刷新 key 池状态时。

注意：Cloudflare Worker 修改变量后通常需要重新部署；重新部署后变量会自动生效。`/admin/reload` 主要用于刷新当前运行实例的内存状态。

## 3. 公开接口说明

| 接口 | 方法 | 说明 |
| --- | --- | --- |
| `/` | `GET` | OMDb 数据 API 代理；根路径有 query 时触发 |
| `/api` | `GET` | OMDb 数据 API 代理别名 |
| `/poster` | `GET` | OMDb poster API 代理 |
| `/docs` | `GET` | 静态文档页面 |
| `/health` | `GET` | 健康检查 |
| `/metrics` | `GET` | 数据看板统计：今日请求数、总请求数 |
| `/admin/stats` | `GET` | 管理统计，需要 `ADMIN_KEY` |
| `/admin/reload` | `POST` | 重载 key，需要 `ADMIN_KEY` |

## 4. Cloudflare Worker 一键部署步骤

### 第 1 步：点击部署按钮

点击本 README 顶部的按钮：

```text
Deploy to Cloudflare
```

或者打开：

```text
https://deploy.workers.cloudflare.com/?url=https://github.com/y08lin4/omdbapi-proxy
```

### 第 2 步：连接 GitHub 并部署

Cloudflare 会克隆本仓库，并根据根目录 `wrangler.toml` 部署 Worker。

### 第 3 步：进入 Worker 设置

部署完成后进入：

```text
Cloudflare Dashboard
→ Workers 和 Pages
→ 你的 Worker
→ 设置
→ 变量和机密
```

### 第 4 步：添加客户端和管理密钥

点击 `添加`，分别添加：

#### CLIENT_KEYS

类型选择：

```text
密钥
```

变量名称：

```text
CLIENT_KEYS
```

值示例：

```text
client_key_1,client_key_2
```

这是你发给 API 用户的 key。用户请求时使用：

```text
?apikey=client_key_1
```

#### ADMIN_KEY

类型选择：

```text
密钥
```

变量名称：

```text
ADMIN_KEY
```

值示例：

```text
admin_xxxxxxxxx
```

这是管理接口用的 key。

### 第 5 步：绑定 KV 并放入 OMDb key 池

因为 Cloudflare 单个环境变量有大小限制，大量 OMDb key 不建议放到 `OMDB_KEYS`。推荐把 OMDb key 池放进 KV。

在 Worker 设置里添加 KV namespace 绑定：

```text
Binding name: STATS_KV
KV namespace: 新建或选择 omdbapi_proxy_stats
```

然后进入该 KV namespace，新增一条记录：

```text
key: omdb:keys
value: omdb_key_1,omdb_key_2,omdb_key_3
```

如果你已经生成了 `omdb_keys_comma.txt`，直接把文件内容整体复制到 `omdb:keys` 的 value 里。

> 小规模 key 池仍可使用 `OMDB_KEYS` 环境变量；当 KV 中存在 `omdb:keys` 时，Worker 会优先使用 KV。

#### 可选：OMDB_KEYS 备用变量

类型选择：

```text
密钥
```

变量名称：

```text
OMDB_KEYS
```

值示例：

```text
omdb_key_1,omdb_key_2
```

仅适合少量 key。大量 key 请使用 KV 的 `omdb:keys`。

### 第 6 步：保存并重新部署

保存变量后，点击页面右下角或顶部的：

```text
部署
```

### 第 7 步：测试

假设 Worker 域名是：

```text
https://omdbapi-proxy123.yourname.workers.dev
```

测试普通请求：

```text
https://omdbapi-proxy123.yourname.workers.dev/?apikey=你的CLIENT_KEY&t=Inception
```

测试健康检查：

```text
https://omdbapi-proxy123.yourname.workers.dev/health
```

测试数据看板接口：

```text
https://omdbapi-proxy123.yourname.workers.dev/metrics
```

测试管理接口：

```text
https://omdbapi-proxy123.yourname.workers.dev/admin/stats?admin_key=你的ADMIN_KEY
```

静态页面：

```text
https://omdbapi-proxy123.yourname.workers.dev/docs
```

## 5. Go 版快速部署

进入 Go 目录：

```bash
cd go
```

复制配置文件：

```bash
cp .env.example .env
cp omdb_keys.example.txt omdb_keys.txt
cp client_keys.example.txt client_keys.txt
```

编辑：

```text
go/omdb_keys.txt      # OMDb 官方 key，一行一个
go/client_keys.txt    # 你发给用户的访问 key，一行一个
go/.env               # 服务配置
```

启动：

```bash
go run .
```

Docker 启动：

```bash
docker compose up -d --build
```

如果你的真实 `omdb_keys.txt` 放在项目根目录，可以在 `go/.env` 中设置：

```env
OMDB_KEYS_FILE=../omdb_keys.txt
```

## 6. 开源安全说明

请不要提交真实 key。仓库已经在 `.gitignore` 中忽略：

```text
.env
.dev.vars
omdb_keys.txt
client_keys.txt
**/omdb_keys.txt
**/client_keys.txt
```

发布前建议运行：

```bash
git status --ignored
```

确认真实 key 文件只出现在 ignored 列表里。

如果真实 key 曾经被 Git 跟踪过，请先执行：

```bash
git rm --cached omdb_keys.txt client_keys.txt
```

## 7. 共同规则

- 没有客户端 key 或 key 错误：直接返回 `401`。
- 客户端 key 不限流。
- OMDb 官方 key 自动轮询。
- 某个 OMDb key 超额、无效、429、5xx 或超时后自动冷却，并尝试下一个 key。
- 普通业务错误，例如 `Movie not found!`，原样返回，不切换 key。
- 静态页面提供数据看板，通过 `/metrics` 显示今日请求数和总请求数。

## Cloudflare KV 持久化统计

> 注意：Cloudflare 一键部署按钮不会替你的账号自动创建 KV namespace。你需要在部署后通过控制台绑定，或在本地登录 Wrangler 后运行下面的脚本创建并绑定。

Cloudflare Worker 版默认使用内存统计，请求数在重新部署、冷启动或切换边缘节点后可能清零。若要让 `/metrics` 的今日请求数和总请求数持久化，可以绑定 Cloudflare KV。

### 控制台配置步骤

1. 进入 `Workers 和 Pages`。
2. 打开你的 Worker。
3. 进入 `设置` → `绑定` 或 `变量和机密` 中的绑定区域。
4. 添加 KV namespace 绑定。
5. 变量名称 / Binding name 填：

```text
STATS_KV
```

6. KV namespace 可以新建，例如：

```text
omdbapi_proxy_stats
```

7. 保存并重新部署 Worker。

绑定完成后，请求统计会写入 KV：

```text
requests:total
requests:day:YYYY-MM-DD
requests:startedAt
requests:lastRequest
```

### Wrangler 配置方式

推荐用项目内置脚本创建并自动写入 `wrangler.toml`：

```bash
# 在仓库根目录执行
npm run cf:kv:create
npm run cf:kv:create-preview
```

如果你在 `cf/` 目录里执行，也可以用：

```bash
cd cf
npm run kv:create
npm run kv:create-preview
```

这些命令会调用 Wrangler 的 `--binding STATS_KV --update-config`，自动创建 KV namespace 并把绑定写入对应的 `wrangler.toml`。

也可以手动创建 KV：

```bash
npx wrangler@latest kv namespace create omdbapi_proxy_stats --binding STATS_KV --update-config
npx wrangler@latest kv namespace create omdbapi_proxy_stats --preview --binding STATS_KV --update-config
```

命令执行后，`wrangler.toml` 会出现类似配置：

```toml
[[kv_namespaces]]
binding = "STATS_KV"
id = "你的生产 KV namespace id"
preview_id = "你的预览 KV namespace id"
```

注意：KV 不是强一致计数器，高并发下可能有轻微计数误差。如果需要严格准确的全局计数，建议后续改用 Durable Objects。

## 8. 统计说明

Go 版当前是内存统计，进程重启后清零。Cloudflare Worker 版如果未绑定 `STATS_KV`，也是内存统计；绑定 `STATS_KV` 后会持久化到 KV。KV 不是强一致计数器，高并发下可能有轻微误差；如果需要严格准确统计，建议后续接入 Durable Objects。

## 许可证

本项目使用 MIT License，见 [LICENSE](LICENSE)。
