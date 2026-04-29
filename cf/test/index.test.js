import assert from "node:assert/strict";
import http from "node:http";
import test from "node:test";
import { once } from "node:events";
import worker, { buildTargetURL, classifyUpstreamFailure, parseKeys } from "../src/index.js";

async function listen(server) {
  server.listen(0, "127.0.0.1");
  await once(server, "listening");
  const { port } = server.address();
  return `http://127.0.0.1:${port}/`;
}

async function close(server) {
  server.close();
  await once(server, "close");
}

function testEnv(baseURL, extra = {}) {
  return {
    CLIENT_KEYS: "client-good",
    OMDB_KEYS: "bad\ngood",
    ADMIN_KEY: "admin-secret",
    OMDB_API_URL: baseURL,
    OMDB_POSTER_URL: new URL("/poster", baseURL).toString(),
    KEY_COOLDOWN: "1m",
    HTTP_TIMEOUT: "2s",
    CORS_ORIGIN: "",
    ...extra
  };
}

test("parseKeys 支持换行、逗号、空格、注释和去重", () => {
  assert.deepEqual(parseKeys("a,b c\n#x\na"), ["a", "b", "c"]);
});

test("buildTargetURL 会替换客户端 apikey", () => {
  const target = buildTargetURL("https://www.omdbapi.com/", "https://proxy.test/?apikey=client&t=Inception", "omdb");
  assert.equal(target, "https://www.omdbapi.com/?t=Inception&apikey=omdb");
});

test("classifyUpstreamFailure 识别额度错误", () => {
  const body = new TextEncoder().encode('{"Response":"False","Error":"Request limit reached!"}');
  const failure = classifyUpstreamFailure(200, body, "application/json");
  assert.equal(failure.retry, true);
  assert.equal(failure.reason, "quota");
});

test("没有客户端 key 禁止请求", async () => {
  const upstream = http.createServer(() => {
    throw new Error("upstream should not be called");
  });
  const base = await listen(upstream);
  try {
    const response = await worker.fetch(new Request("https://proxy.test/?t=Inception"), testEnv(base));
    assert.equal(response.status, 401);
    assert.deepEqual(await response.json(), { Response: "False", Error: "Invalid API key." });
  } finally {
    await close(upstream);
  }
});

test("OMDb key 额度错误时自动切换下一个 key", async () => {
  const seenKeys = [];
  const upstream = http.createServer((req, res) => {
    const url = new URL(req.url, "http://upstream.test");
    const key = url.searchParams.get("apikey");
    seenKeys.push(key);
    res.setHeader("content-type", "application/json");
    if (key === "bad") {
      res.end(JSON.stringify({ Response: "False", Error: "Request limit reached!" }));
      return;
    }
    res.end(JSON.stringify({ Response: "True", Title: url.searchParams.get("t"), UsedKey: key, ClientKeyWasForwarded: key === "client-good" }));
  });
  const base = await listen(upstream);
  try {
    const response = await worker.fetch(new Request("https://proxy.test/?apikey=client-good&t=Inception"), testEnv(base));
    assert.equal(response.status, 200);
    const json = await response.json();
    assert.equal(json.Title, "Inception");
    assert.equal(json.UsedKey, "good");
    assert.equal(json.ClientKeyWasForwarded, false);
    assert.deepEqual(seenKeys, ["bad", "good"]);
  } finally {
    await close(upstream);
  }
});

test("普通业务错误不切换 key", async () => {
  let calls = 0;
  const upstream = http.createServer((req, res) => {
    calls += 1;
    res.setHeader("content-type", "application/json");
    res.end(JSON.stringify({ Response: "False", Error: "Movie not found!" }));
  });
  const base = await listen(upstream);
  try {
    const response = await worker.fetch(new Request("https://proxy.test/?apikey=client-good&t=NoSuchMovie"), testEnv(base, { OMDB_KEYS: "k1\nk2" }));
    assert.equal(response.status, 200);
    assert.deepEqual(await response.json(), { Response: "False", Error: "Movie not found!" });
    assert.equal(calls, 1);
  } finally {
    await close(upstream);
  }
});
