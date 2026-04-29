export const DOCS_HTML = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>OMDb API Proxy</title>
  <style>
    :root { color-scheme: light dark; --bg:#f6f7fb; --card:#fff; --text:#111827; --muted:#6b7280; --line:#e5e7eb; --code:#f3f4f6; --accent:#2563eb; }
    @media (prefers-color-scheme: dark) { :root { --bg:#0b1020; --card:#111827; --text:#f9fafb; --muted:#9ca3af; --line:#1f2937; --code:#0f172a; --accent:#60a5fa; } }
    * { box-sizing: border-box; }
    body { margin:0; background:var(--bg); color:var(--text); font-family:"Noto Sans SC","Microsoft YaHei","PingFang SC","Hiragino Sans GB",Arial,sans-serif; line-height:1.6; }
    main { width:min(980px, calc(100% - 32px)); margin:0 auto; padding:48px 0; }
    header { margin-bottom:24px; }
    h1 { margin:0 0 8px; font-size:clamp(30px,5vw,48px); letter-spacing:-.04em; }
    h2 { margin:0 0 14px; font-size:20px; }
    p { margin:0; color:var(--muted); }
    .grid { display:grid; grid-template-columns:repeat(2,minmax(0,1fr)); gap:16px; margin:24px 0; }
    .card { background:var(--card); border:1px solid var(--line); border-radius:18px; padding:20px; box-shadow:0 10px 30px rgba(15,23,42,.06); }
    .stat-label { color:var(--muted); font-size:14px; }
    .stat-value { margin-top:8px; font-size:clamp(36px,8vw,64px); font-weight:800; line-height:1; letter-spacing:-.05em; }
    .examples { display:grid; gap:12px; }
    .example { border:1px solid var(--line); border-radius:14px; overflow:hidden; background:var(--card); }
    .example-title { padding:12px 14px; border-bottom:1px solid var(--line); color:var(--muted); font-size:14px; }
    pre { margin:0; padding:14px; overflow:auto; background:var(--code); }
    code { font-family:ui-monospace,SFMono-Regular,Menlo,Consolas,"Liberation Mono",monospace; font-size:13px; white-space:pre; }
    .note { margin-top:14px; font-size:14px; color:var(--muted); }
    .badge { display:inline-flex; align-items:center; gap:6px; margin-top:14px; padding:6px 10px; border:1px solid var(--line); border-radius:999px; color:var(--muted); font-size:13px; }
    .dot { width:8px; height:8px; border-radius:999px; background:var(--accent); }
    @media (max-width:640px) { .grid { grid-template-columns:1fr; } main { padding:32px 0; } }
  </style>
</head>
<body>
  <main>
    <header>
      <h1>OMDb API Proxy</h1>
      <p>&#31616;&#27905;&#30340; OMDb &#22810; Key &#20195;&#29702;&#26381;&#21153;&#12290;&#35831;&#27714;&#26102;&#20351;&#29992;&#20320;&#30340; <code>CLIENT_KEYS</code>&#65292;&#26381;&#21153;&#31471;&#20250;&#33258;&#21160;&#20999;&#25442;&#20869;&#37096; OMDb Key&#12290;</p>
      <div class="badge"><span class="dot"></span><span id="storage">&#32479;&#35745;&#21152;&#36733;&#20013;</span></div>
    </header>
    <section class="grid" aria-label="&#35831;&#27714;&#32479;&#35745;">
      <div class="card"><div class="stat-label">&#20170;&#26085;&#35843;&#29992;&#27425;&#25968;</div><div class="stat-value" id="stats-today">--</div></div>
      <div class="card"><div class="stat-label">&#24635;&#35843;&#29992;&#27425;&#25968;</div><div class="stat-value" id="stats-total">--</div></div>
    </section>
    <section class="card">
      <h2>&#35831;&#27714;&#31034;&#20363;</h2>
      <p class="note">&#20197;&#19979;&#31034;&#20363;&#20351;&#29992;&#29983;&#20135;&#22495;&#21517; <strong>https://omdbapi.ailinyu.de</strong>&#12290;&#35831;&#25226; <code>YOUR_CLIENT_KEY</code> &#26367;&#25442;&#25104;&#20320;&#30340;&#23458;&#25143;&#31471;&#35775;&#38382; Key&#12290;</p>
      <div class="examples" style="margin-top:16px">
        <div class="example"><div class="example-title">&#25353;&#26631;&#39064;&#26597;&#35810;</div><pre><code>https://omdbapi.ailinyu.de/?apikey=YOUR_CLIENT_KEY&amp;t=Inception</code></pre></div>
        <div class="example"><div class="example-title">&#25628;&#32034;&#30005;&#24433;</div><pre><code>https://omdbapi.ailinyu.de/?apikey=YOUR_CLIENT_KEY&amp;s=Batman&amp;page=2</code></pre></div>
        <div class="example"><div class="example-title">&#25353; IMDb ID &#26597;&#35810;</div><pre><code>https://omdbapi.ailinyu.de/?apikey=YOUR_CLIENT_KEY&amp;i=tt1375666</code></pre></div>
        <div class="example"><div class="example-title">&#23436;&#25972;&#21095;&#24773;</div><pre><code>https://omdbapi.ailinyu.de/?apikey=YOUR_CLIENT_KEY&amp;t=Inception&amp;plot=full</code></pre></div>
        <div class="example"><div class="example-title">Poster API</div><pre><code>https://omdbapi.ailinyu.de/poster?apikey=YOUR_CLIENT_KEY&amp;i=tt1375666</code></pre></div>
        <div class="example"><div class="example-title">&#31649;&#29702;&#32479;&#35745;</div><pre><code>https://omdbapi.ailinyu.de/admin/stats?admin_key=YOUR_ADMIN_KEY</code></pre></div>
      </div>
    </section>
  </main>
  <script>
    async function loadStats() {
      try {
        const res = await fetch('/metrics', { cache: 'no-store' });
        if (!res.ok) throw new Error('HTTP ' + res.status);
        const data = await res.json();
        const stats = data.requests || {};
        document.getElementById('stats-today').textContent = String(stats.today ?? 0);
        document.getElementById('stats-total').textContent = String(stats.total ?? 0);
        document.getElementById('storage').textContent = '\u7edf\u8ba1\u5b58\u50a8\uff1a' + (stats.storage || 'memory');
      } catch (error) {
        document.getElementById('storage').textContent = '\u7edf\u8ba1\u52a0\u8f7d\u5931\u8d25';
      }
    }
    loadStats();
  </script>
</body>
</html>
`;
