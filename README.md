# OMDb API 管理器：Go 版 + Cloudflare Worker 版

本项目提供两个部署版本：

```text
go/   # Go 公网部署版：适合 VPS、Docker、自建服务器；从 txt 文件读取 key
cf/   # Cloudflare Worker 版：适合边缘部署；从 Worker 环境变量 / Secrets 读取 key
```

两个版本的外部请求格式一致：

```text
GET /?apikey=YOUR_CLIENT_KEY&t=Inception
GET /?apikey=YOUR_CLIENT_KEY&s=Batman&page=2
GET /?apikey=YOUR_CLIENT_KEY&i=tt1375666
GET /poster?apikey=YOUR_CLIENT_KEY&i=tt1375666
```

这里的 `apikey` 是你发给调用方的服务 key，不是 OMDb 官方 key。服务端或 Worker 会自动替换成内部 OMDb 官方 key。

## 开源安全说明

请不要提交真实 key。仓库已经在 `.gitignore` 中忽略：

```text
.env
.dev.vars
omdb_keys.txt
client_keys.txt
**/omdb_keys.txt
**/client_keys.txt
```

如果你已经把真实 key 放在项目根目录的 `omdb_keys.txt`，可以继续本地使用；该文件默认不会进入 Git。发布前建议运行：

```bash
git status --ignored
```

确认真实 key 文件只出现在 ignored 列表里。

如果真实 key 曾经被 Git 跟踪过，请先执行：

```bash
git rm --cached omdb_keys.txt client_keys.txt
```

## Go 版部署

适合 VPS / Docker / 自建服务器。

```powershell
cd go
Copy-Item .env.example .env
Copy-Item omdb_keys.example.txt omdb_keys.txt
Copy-Item client_keys.example.txt client_keys.txt
go run .
```

如果你的真实 `omdb_keys.txt` 放在项目根目录，可以复制到 `go/omdb_keys.txt`，也可以在 `go/.env` 中设置：

```env
OMDB_KEYS_FILE=../omdb_keys.txt
```

详细文档见：[`go/README.md`](go/README.md)

## Cloudflare Worker 版部署

适合不想维护服务器的场景。Worker 不能读取你的服务器本地 `omdb_keys.txt`，需要把一行一个 key 的内容粘贴到 Cloudflare Secret `OMDB_KEYS` 中。

本地调试：

```bash
cd cf
cp .dev.vars.example .dev.vars
npm test
npm run dev
```

线上部署：

```bash
cd cf
npx wrangler@latest secret put CLIENT_KEYS
npx wrangler@latest secret put OMDB_KEYS
npx wrangler@latest secret put ADMIN_KEY
npx wrangler@latest deploy
```

详细文档见：[`cf/README.md`](cf/README.md)

## Cloudflare 一键部署

仓库发布到 GitHub 后，可以在 README 中放置下面的按钮。把链接里的仓库地址替换成你的真实仓库地址：

```md
[![部署到 Cloudflare](https://deploy.workers.cloudflare.com/button)](https://deploy.workers.cloudflare.com/?url=https://github.com/你的用户名/你的仓库)
```

> 注意：一键部署不会把真实 key 写进仓库。部署过程中或部署后，需要在 Cloudflare 控制台 / Wrangler 中配置 `CLIENT_KEYS`、`OMDB_KEYS`、`ADMIN_KEY`。

## 共同规则

- 没有客户端 key 或 key 错误：直接返回 `401`。
- 客户端 key 不限流。
- OMDb 官方 key 自动轮询。
- 某个 OMDb key 超额、无效、429、5xx 或超时后自动冷却，并尝试下一个 key。
- 普通业务错误，例如 `Movie not found!`，原样返回，不切换 key。


## 许可证

本项目使用 MIT License，见 [LICENSE](LICENSE)。

