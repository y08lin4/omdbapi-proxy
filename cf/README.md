# OMDb API 管理器 - Cloudflare Worker 版

[![Deploy to Cloudflare](https://deploy.workers.cloudflare.com/button)](https://deploy.workers.cloudflare.com/?url=https://github.com/y08lin4/omdbapi-proxy)

这个目录是 Cloudflare Worker 版本。客户端使用你发放的 `apikey` 请求，Worker 内部从 OMDb 官方 key 池中选择 key 并替换转发。

## 1. 变量说明

需要设置三个密钥变量：

| 变量名 | 用途 | 示例 |
| --- | --- | --- |
| `CLIENT_KEYS` | 你发给调用方的访问 key | `client_key_1,client_key_2` |
| `OMDB_KEYS` | OMDb 官方 key 池 | `omdb_key_1,omdb_key_2` |
| `ADMIN_KEY` | 管理接口 key | `admin_xxxxx` |

`CLIENT_KEYS` 和 `OMDB_KEYS` 支持逗号、空格或换行分隔。

## 2. Cloudflare 控制台部署步骤

### 第 1 步：点击一键部署按钮

点击上方 `Deploy to Cloudflare` 按钮，或者打开：

```text
https://deploy.workers.cloudflare.com/?url=https://github.com/y08lin4/omdbapi-proxy
```

### 第 2 步：部署 Worker

根据 Cloudflare 页面提示连接 GitHub 并部署。

### 第 3 步：进入变量设置

```text
Cloudflare Dashboard
→ Workers 和 Pages
→ 你的 Worker
→ 设置
→ 变量和机密
```

### 第 4 步：添加 CLIENT_KEYS

- 类型：`密钥`
- 变量名称：`CLIENT_KEYS`
- 值：你发给用户的访问 key

示例：

```text
client_key_1,client_key_2
```

用户请求时使用：

```text
?apikey=client_key_1
```

### 第 5 步：添加 OMDB_KEYS

- 类型：`密钥`
- 变量名称：`OMDB_KEYS`
- 值：OMDb 官方 key 池

示例：

```text
omdb_key_1,omdb_key_2,omdb_key_3
```

如果你有 `omdb_keys.txt`，可以把它转换成逗号分隔后粘贴进去。

### 第 6 步：添加 ADMIN_KEY

- 类型：`密钥`
- 变量名称：`ADMIN_KEY`
- 值：管理接口 key

示例：

```text
admin_xxxxxxxxx
```

### 第 7 步：保存并重新部署

保存变量后点击：

```text
部署
```

## 3. 请求示例

假设 Worker 域名为：

```text
https://example.yourname.workers.dev
```

按标题查询：

```text
https://example.yourname.workers.dev/?apikey=CLIENT_KEY&t=Inception
```

搜索：

```text
https://example.yourname.workers.dev/?apikey=CLIENT_KEY&s=Batman&page=2
```

按 IMDb ID 查询：

```text
https://example.yourname.workers.dev/?apikey=CLIENT_KEY&i=tt1375666
```

Poster API：

```text
https://example.yourname.workers.dev/poster?apikey=CLIENT_KEY&i=tt1375666
```

静态页面：

```text
https://example.yourname.workers.dev/docs
```

## 4. 管理接口

管理接口只给管理员使用，需要 `ADMIN_KEY`。

### 查看统计

```text
GET /admin/stats?admin_key=ADMIN_KEY
```

返回内容包括：

- 客户端 key 数量。
- OMDb key 总数。
- 可用 OMDb key 数量。
- 每个 OMDb key 的脱敏值、请求次数、成功次数、失败次数、冷却状态。
- 今日请求数和总请求数。

### 重载 key

```text
POST /admin/reload?admin_key=ADMIN_KEY
```

作用：重新解析当前 Worker 环境变量里的 `CLIENT_KEYS` 和 `OMDB_KEYS`，并刷新内存 key 池。

注意：如果你在 Cloudflare 控制台修改了变量，通常还需要重新部署 Worker。

## 5. 公开状态接口

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

## 6. 本地开发

```bash
cp .dev.vars.example .dev.vars
npm test
npm run dev
```

`.dev.vars` 示例：

```env
CLIENT_KEYS=local_client_key
OMDB_KEYS=your_omdb_key_1,your_omdb_key_2
ADMIN_KEY=local_admin_key
```

## Cloudflare KV 持久化统计

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

也可以用命令创建 KV：

```bash
npx wrangler@latest kv namespace create omdbapi_proxy_stats
npx wrangler@latest kv namespace create omdbapi_proxy_stats --preview
```

然后把返回的 `id` 和 `preview_id` 填到根目录 `wrangler.toml`：

```toml
[[kv_namespaces]]
binding = "STATS_KV"
id = "你的生产 KV namespace id"
preview_id = "你的预览 KV namespace id"
```

注意：KV 不是强一致计数器，高并发下可能有轻微计数误差。如果需要严格准确的全局计数，建议后续改用 Durable Objects。

## 7. 注意

Cloudflare Worker 的 key 轮询和冷却状态仍是每个 isolate / 边缘节点本地的，不保证全局一致。请求统计如果绑定了 `STATS_KV` 会持久化到 KV；未绑定时仍是内存统计。如果需要严格准确的全局计数和冷却状态，建议后续改用 Durable Objects。
