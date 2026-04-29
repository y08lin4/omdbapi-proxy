# OMDb API 管理器 - Cloudflare Worker 版

这个目录是 Cloudflare Worker 版本。它和 Go 版保持同样的外部请求格式：客户端使用你发放的 `apikey` 请求，Worker 内部从 OMDb 官方 key 池中选择 key 并替换转发。

参考 Cloudflare Worker 官方文档：

- Workers 总览：https://developers.cloudflare.com/workers/
- Fetch handler：https://developers.cloudflare.com/workers/runtime-apis/handlers/fetch/
- Wrangler CLI：https://developers.cloudflare.com/workers/wrangler/
- Deploy Button：https://developers.cloudflare.com/workers/tutorials/deploy-button/

## 路由

- `GET /` 或 `GET /api`：OMDb 数据 API 代理。
- `GET /poster`：OMDb poster API 代理。
- `GET /docs`：静态文档页。
- `GET /health`：健康检查。
- `GET /admin/stats?admin_key=ADMIN_KEY`：查看 key 池状态。
- `POST /admin/reload?admin_key=ADMIN_KEY`：重新解析环境变量中的 key，并重置内存状态。

## 支持的 OMDb 参数

Worker 不限制参数名，除 `apikey` 外全部原样透传。常用官方参数包括：

- `i`：IMDb ID
- `t`：标题
- `s`：搜索关键词
- `type`：`movie` / `series` / `episode`
- `y`：年份
- `plot`：`short` / `full`
- `r`：`json` / `xml`
- `callback`：JSONP 回调
- `v`：API 版本
- `page`：搜索分页
- `Season` / `Episode`：剧集季 / 集

## 本地开发

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

请求：

```text
http://127.0.0.1:8787/?apikey=local_client_key&t=Inception
http://127.0.0.1:8787/?apikey=local_client_key&s=Batman&page=2
http://127.0.0.1:8787/poster?apikey=local_client_key&i=tt1375666
```

## 线上部署

不要把真实 key 写进 `wrangler.toml`。线上使用 Cloudflare Secrets：

```bash
npx wrangler@latest secret put CLIENT_KEYS
npx wrangler@latest secret put OMDB_KEYS
npx wrangler@latest secret put ADMIN_KEY
npx wrangler@latest deploy
```

`CLIENT_KEYS` 和 `OMDB_KEYS` 都可以粘贴为多行，一行一个 key；也支持逗号或空格分隔。

## 一键部署按钮

仓库发布到 GitHub 后，可以在根目录 README 或本文件中加入：

```md
[![部署到 Cloudflare](https://deploy.workers.cloudflare.com/button)](https://deploy.workers.cloudflare.com/?url=https://github.com/你的用户名/你的仓库/tree/main/cf)
```

如果你把 `cf/` 单独作为一个仓库，链接可以改成：

```md
[![部署到 Cloudflare](https://deploy.workers.cloudflare.com/button)](https://deploy.workers.cloudflare.com/?url=https://github.com/你的用户名/你的仓库)
```

部署完成后，在 Cloudflare 控制台或 Wrangler 中设置：

- `CLIENT_KEYS`：你发给调用方的访问 key。
- `OMDB_KEYS`：OMDb 官方 key 池，一行一个。
- `ADMIN_KEY`：管理接口 key。

## 注意

Cloudflare Worker 的内存状态是每个 isolate / 边缘节点本地的，不保证全局一致。因此当前版本的轮询、冷却和统计是“边缘本地状态”。如果你需要跨全球节点统一冷却或统计，需要再加 Durable Objects 或 KV。
