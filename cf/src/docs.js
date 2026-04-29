export const DOCS_HTML = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>OMDb API Worker 代理文档</title>
  <style>
    :root { color-scheme: light dark; --bg:#0f172a; --card:#111827; --text:#e5e7eb; --muted:#94a3b8; --line:#334155; --accent:#38bdf8; --code:#020617; }
    @media (prefers-color-scheme: light) { :root { --bg:#f8fafc; --card:#ffffff; --text:#0f172a; --muted:#475569; --line:#e2e8f0; --accent:#0284c7; --code:#f1f5f9; } }
    * { box-sizing: border-box; }
    body { margin:0; font-family: "Noto Sans SC", "Microsoft YaHei", "PingFang SC", "Hiragino Sans GB", "Source Han Sans SC", Arial, sans-serif; background:var(--bg); color:var(--text); line-height:1.6; }
    header { padding:42px 22px 24px; border-bottom:1px solid var(--line); background:linear-gradient(135deg, rgba(56,189,248,.18), transparent 55%); }
    main,.wrap { width:min(1100px, 100%); margin:0 auto; }
    main { padding:24px 18px 60px; }
    h1 { margin:0 0 8px; font-size:clamp(28px, 4vw, 44px); }
    h2 { margin-top:32px; padding-top:12px; border-top:1px solid var(--line); }
    h3 { margin-top:22px; }
    p, li { color:var(--muted); }
    a { color:var(--accent); }
    .card { background:var(--card); border:1px solid var(--line); border-radius:16px; padding:18px; margin:16px 0; box-shadow:0 10px 30px rgba(0,0,0,.12); }
    code, pre { font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, "Liberation Mono", monospace; }
    code { background:var(--code); padding:.15rem .35rem; border-radius:6px; }
    pre { background:var(--code); padding:14px; border-radius:12px; overflow:auto; border:1px solid var(--line); }
    table { width:100%; border-collapse:collapse; overflow:hidden; border-radius:12px; border:1px solid var(--line); }
    th, td { padding:10px 12px; border-bottom:1px solid var(--line); text-align:left; vertical-align:top; }
    th { color:var(--text); background:rgba(148,163,184,.12); }
    tr:last-child td { border-bottom:none; }
  </style>
</head>
<body>
  <header>
    <div class="wrap">
      <h1>OMDb API Worker 代理文档</h1>
      <p>本服务兼容 OMDb 官方查询格式，但 <code>apikey</code> 使用本站发放的访问 key。Worker 会自动替换为内部 OMDb 官方 key。</p>
    </div>
  </header>
  <main>
    <section class="card">
      <h2>基础规则</h2>
      <ul>
        <li>所有代理 API 必须带 <code>?apikey=YOUR_CLIENT_KEY</code>；没有 key 或 key 错误返回 <code>401</code>。</li>
        <li><code>apikey</code> 是本站服务 key，不是 OMDb 官方 key。</li>
        <li>也支持 <code>X-API-Key</code> 与 <code>Authorization: Bearer YOUR_CLIENT_KEY</code>。</li>
        <li>除 <code>apikey</code> 外，其余 query 参数原样转发给 OMDb。</li>
      </ul>
      <pre><code>GET /?apikey=YOUR_CLIENT_KEY&amp;t=Inception</code></pre>
    </section>

    <section class="card">
      <h2>接口总览</h2>
      <pre><code>GET  /                 数据 API，根路径有 query 时触发
GET  /api              数据 API 别名
GET  /poster           Poster API
GET  /docs             本文档
GET  /health           健康检查
GET  /admin/stats      管理统计，需要 admin_key
POST /admin/reload     重载环境变量中的 key 状态，需要 admin_key</code></pre>
    </section>

    <section class="card">
      <h2>按标题 / IMDb ID 查询</h2>
      <table>
        <tr><th>参数</th><th>说明</th></tr>
        <tr><td><code>i</code></td><td>IMDb ID，例如 <code>tt1375666</code></td></tr>
        <tr><td><code>t</code></td><td>标题</td></tr>
        <tr><td><code>type</code></td><td><code>movie</code> / <code>series</code> / <code>episode</code></td></tr>
        <tr><td><code>y</code></td><td>年份</td></tr>
        <tr><td><code>plot</code></td><td><code>short</code> / <code>full</code></td></tr>
        <tr><td><code>r</code></td><td><code>json</code> / <code>xml</code></td></tr>
        <tr><td><code>callback</code></td><td>JSONP 回调</td></tr>
        <tr><td><code>v</code></td><td>API 版本</td></tr>
      </table>
      <pre><code>GET /?apikey=YOUR_CLIENT_KEY&amp;t=Inception&amp;plot=full
GET /?apikey=YOUR_CLIENT_KEY&amp;i=tt1375666
GET /?apikey=YOUR_CLIENT_KEY&amp;t=Inception&amp;r=xml</code></pre>
    </section>

    <section class="card">
      <h2>搜索</h2>
      <table>
        <tr><th>参数</th><th>说明</th></tr>
        <tr><td><code>s</code></td><td>搜索关键词</td></tr>
        <tr><td><code>type</code></td><td>结果类型</td></tr>
        <tr><td><code>y</code></td><td>年份</td></tr>
        <tr><td><code>page</code></td><td>分页，官方范围通常为 <code>1-100</code></td></tr>
      </table>
      <pre><code>GET /?apikey=YOUR_CLIENT_KEY&amp;s=Batman
GET /?apikey=YOUR_CLIENT_KEY&amp;s=Batman&amp;type=movie&amp;page=2</code></pre>
    </section>

    <section class="card">
      <h2>剧集 Season / Episode</h2>
      <pre><code>GET /?apikey=YOUR_CLIENT_KEY&amp;t=Game%20of%20Thrones&amp;Season=1
GET /?apikey=YOUR_CLIENT_KEY&amp;t=Game%20of%20Thrones&amp;Season=1&amp;Episode=1</code></pre>
    </section>

    <section class="card">
      <h2>Poster API</h2>
      <p>可用性取决于你的 OMDb 官方 key 权限。</p>
      <pre><code>GET /poster?apikey=YOUR_CLIENT_KEY&amp;i=tt1375666
GET /poster?apikey=YOUR_CLIENT_KEY&amp;i=tt1375666&amp;h=600</code></pre>
    </section>
  </main>
</body>
</html>`;
