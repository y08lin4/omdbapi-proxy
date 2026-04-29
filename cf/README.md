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

## 7. 注意

Cloudflare Worker 的内存状态是每个 isolate / 边缘节点本地的，不保证全局一致。因此当前版本的轮询、冷却和统计是“边缘本地状态”。如果你需要跨全球节点统一冷却或统计，需要再加 Durable Objects、KV 或 D1。
