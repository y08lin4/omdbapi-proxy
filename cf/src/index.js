import { DOCS_HTML } from "./docs.js";
const DEFAULT_OMDB_API_URL = "https://www.omdbapi.com/";
const DEFAULT_OMDB_POSTER_URL = "https://img.omdbapi.com/";
const DEFAULT_HTTP_TIMEOUT_MS = 10_000;
const DEFAULT_KEY_COOLDOWN_MS = 5 * 60 * 1000;
const RETRYABLE_STATUS = new Set([408, 425, 429, 500, 502, 503, 504]);
const HOP_BY_HOP_HEADERS = new Set([
  "connection",
  "keep-alive",
  "proxy-authenticate",
  "proxy-authorization",
  "te",
  "trailer",
  "transfer-encoding",
  "upgrade"
]);
const JSONP_CALLBACK_RE = /^[A-Za-z_$][0-9A-Za-z_$]*(\.[A-Za-z_$][0-9A-Za-z_$]*)*$/;

let runtimeState;

export default {
  async fetch(request, env = {}, ctx = {}) {
    return handleRequest(request, env, ctx);
  }
};

export async function handleRequest(request, env = {}, ctx = {}) {
  const state = getRuntimeState(env);
  const url = new URL(request.url);
  const pathname = normalizePath(url.pathname);

  if (request.method === "OPTIONS") {
    return optionsResponse(env, request);
  }

  try {
    if (pathname === "/" && url.search === "") {
      return docsResponse(env, request);
    }

    switch (pathname) {
      case "/":
      case "/api":
        return proxyRequest(request, env, state, env.OMDB_API_URL || DEFAULT_OMDB_API_URL);
      case "/poster":
        return proxyRequest(request, env, state, env.OMDB_POSTER_URL || DEFAULT_OMDB_POSTER_URL);
      case "/docs":
      case "/index.html":
        return docsResponse(env, request);
      case "/health":
        return jsonResponse(env, request, 200, {
          ok: state.omdbKeys.size() > 0 && state.clients.size > 0,
          service: "omdb-api-manager-worker",
          runtime: "cloudflare-workers",
          clientKeyCount: state.clients.size,
          omdb: state.omdbKeys.stats(false)
        });
      case "/admin/stats":
        return adminStats(request, env, state);
      case "/admin/reload":
        return adminReload(request, env);
      default:
        return omdbErrorResponse(env, request, 404, "Not found. Use /, /api, /poster, /docs, /health, /admin/stats or /admin/reload.");
    }
  } catch (error) {
    console.error("worker error", error && (error.stack || error.message || error));
    return omdbErrorResponse(env, request, 500, "Internal server error.");
  }
}

async function proxyRequest(request, env, state, upstreamBaseURL) {
  if (request.method !== "GET" && request.method !== "HEAD") {
    return omdbErrorResponse(env, request, 405, "Method not allowed. OMDb-compatible requests use GET.");
  }

  const clientKey = clientAPIKey(request);
  if (!state.clients.has(clientKey)) {
    return omdbErrorResponse(env, request, 401, "Invalid API key.");
  }

  if (state.omdbKeys.size() === 0) {
    return omdbErrorResponse(env, request, 503, "No upstream OMDb API keys configured.");
  }

  let maxAttempts = Number.parseInt(env.MAX_ATTEMPTS_PER_REQUEST || "0", 10);
  if (!Number.isFinite(maxAttempts) || maxAttempts <= 0 || maxAttempts > state.omdbKeys.size()) {
    maxAttempts = state.omdbKeys.size();
  }

  const tried = new Set();
  const attemptErrors = [];
  for (let attempt = 1; attempt <= maxAttempts; attempt += 1) {
    const selected = state.omdbKeys.acquire(tried);
    if (!selected) break;
    tried.add(selected.index);

    const targetURL = buildTargetURL(upstreamBaseURL, request.url, selected.value);
    try {
      const upstream = await fetchWithTimeout(targetURL, request, durationToMs(env.HTTP_TIMEOUT || env.HTTP_TIMEOUT_MS, DEFAULT_HTTP_TIMEOUT_MS));
      const body = new Uint8Array(await upstream.arrayBuffer());
      const failure = classifyUpstreamFailure(upstream.status, body, upstream.headers.get("content-type") || "");

      if (failure.retry) {
        state.omdbKeys.reportFailure(selected, failure.reason);
        state.omdbKeys.release(selected);
        attemptErrors.push({ attempt, keyIndex: selected.index, key: selected.masked, reason: failure.reason, message: failure.message });
        continue;
      }

      state.omdbKeys.reportSuccess(selected);
      state.omdbKeys.release(selected);
      return upstreamResponse(env, request, upstream, body, selected, attempt);
    } catch (error) {
      const reason = error && error.name === "AbortError" ? "timeout" : "network_error";
      state.omdbKeys.reportFailure(selected, reason);
      state.omdbKeys.release(selected);
      attemptErrors.push({ attempt, keyIndex: selected.index, key: selected.masked, reason, message: String(error && error.message || error) });
    }
  }

  if (attemptErrors.length) console.warn("all attempted OMDb keys failed", JSON.stringify(attemptErrors));
  return omdbErrorResponse(env, request, 503, "All configured OMDb API keys failed or are cooling down.");
}

function getRuntimeState(env) {
  const cooldownMs = durationToMs(env.KEY_COOLDOWN || env.KEY_COOLDOWN_MS, DEFAULT_KEY_COOLDOWN_MS);
  const signature = stateSignature(env, cooldownMs);

  if (!runtimeState || runtimeState.signature !== signature) {
    runtimeState = new RuntimeState(parseKeys(env.OMDB_KEYS || ""), parseKeys(env.CLIENT_KEYS || ""), cooldownMs, signature);
  }
  return runtimeState;
}

function stateSignature(env, cooldownMs) {
  return JSON.stringify([env.OMDB_KEYS || "", env.CLIENT_KEYS || "", cooldownMs]);
}

export class RuntimeState {
  constructor(omdbKeys, clientKeys, cooldownMs, signature = "") {
    this.signature = signature;
    this.omdbKeys = new KeyPool(omdbKeys, cooldownMs);
    this.clients = new Set(clientKeys);
  }
}

export class KeyPool {
  constructor(keys, cooldownMs = DEFAULT_KEY_COOLDOWN_MS) {
    this.cooldownMs = cooldownMs;
    this.cursor = 0;
    this.keys = [];
    this.reload(keys);
  }

  reload(keys) {
    const old = new Map(this.keys.map((key) => [key.value, key]));
    this.keys = keys.map((value, index) => {
      const previous = old.get(value);
      if (previous) return { ...previous, index, masked: maskKey(value) };
      return {
        index,
        value,
        masked: maskKey(value),
        inFlight: 0,
        total: 0,
        successes: 0,
        failures: 0,
        disabledUntil: 0,
        lastError: "",
        lastUsedAt: 0
      };
    });
    if (this.keys.length === 0) this.cursor = 0;
    else this.cursor %= this.keys.length;
  }

  size() {
    return this.keys.length;
  }

  acquire(tried = new Set()) {
    const now = Date.now();
    if (this.keys.length === 0) return null;

    for (let step = 0; step < this.keys.length; step += 1) {
      const index = (this.cursor + step) % this.keys.length;
      const key = this.keys[index];
      if (tried.has(index)) continue;
      if (key.disabledUntil > now) continue;

      this.cursor = (index + 1) % this.keys.length;
      key.inFlight += 1;
      key.total += 1;
      key.lastUsedAt = now;
      return key;
    }
    return null;
  }

  release(key) {
    if (!key) return;
    key.inFlight = Math.max(0, key.inFlight - 1);
  }

  reportSuccess(key) {
    if (!key) return;
    key.successes += 1;
    key.lastError = "";
  }

  reportFailure(key, reason) {
    if (!key) return;
    key.failures += 1;
    key.lastError = reason;
    key.disabledUntil = Date.now() + this.cooldownMs;
  }

  stats(includeKeys) {
    const now = Date.now();
    const stats = {
      totalKeys: this.keys.length,
      availableKeys: this.keys.filter((key) => key.disabledUntil <= now).length,
      cooldownMs: this.cooldownMs
    };
    if (includeKeys) {
      stats.keys = this.keys.map((key) => ({
        index: key.index,
        key: key.masked,
        inFlight: key.inFlight,
        total: key.total,
        successes: key.successes,
        failures: key.failures,
        available: key.disabledUntil <= now,
        disabledForMs: Math.max(0, key.disabledUntil - now),
        lastError: key.lastError || undefined,
        lastUsedAt: key.lastUsedAt ? new Date(key.lastUsedAt).toISOString() : undefined
      }));
    }
    return stats;
  }
}

export function parseKeys(value) {
  const seen = new Set();
  const keys = [];
  const lines = String(value || "").replace(/^\uFEFF/, "").split(/\r?\n/);

  for (const rawLine of lines) {
    const line = rawLine.trim();
    if (!line || line.startsWith("#")) continue;
    for (const part of line.split(/[\s,]+/)) {
      const key = part.trim();
      if (!key || key.startsWith("#") || seen.has(key)) continue;
      seen.add(key);
      keys.push(key);
    }
  }
  return keys;
}

export function buildTargetURL(upstreamBaseURL, incomingURL, upstreamAPIKey) {
  const incoming = new URL(incomingURL);
  const target = new URL(upstreamBaseURL);
  target.search = "";

  for (const [name, value] of incoming.searchParams.entries()) {
    if (name.toLowerCase() === "apikey") continue;
    target.searchParams.append(name, value);
  }
  target.searchParams.set("apikey", upstreamAPIKey);
  return target.toString();
}

async function fetchWithTimeout(targetURL, originalRequest, timeoutMs) {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), timeoutMs);
  try {
    const headers = new Headers();
    headers.set("Accept", originalRequest.headers.get("Accept") || "*/*");
    headers.set("User-Agent", originalRequest.headers.get("User-Agent") || "omdb-api-manager-worker/1.0");
    return await fetch(targetURL, { method: "GET", headers, signal: controller.signal });
  } finally {
    clearTimeout(timeout);
  }
}

export function classifyUpstreamFailure(statusCode, bodyBytes, contentType = "") {
  if (RETRYABLE_STATUS.has(statusCode)) {
    return { retry: true, reason: `http_${statusCode}`, message: statusText(statusCode) };
  }

  const body = bytesToText(bodyBytes, 1024 * 1024);
  if (!body) return { retry: false };
  const lower = body.toLowerCase();
  const message = extractOMDBErrorMessage(body, contentType);

  if (/request\s+limit|daily\s+limit|limit\s+reached|quota|exceed/.test(lower)) {
    return { retry: true, reason: "quota", message };
  }
  if (/invalid\s+(api\s*)?key|invalid\s+apikey|no\s+(api\s*)?key|api\s*key\s+required/.test(lower)) {
    return { retry: true, reason: "invalid_key", message };
  }
  return { retry: false };
}

function extractOMDBErrorMessage(body, contentType = "") {
  const trimmed = String(body || "").trim();
  if (!trimmed) return "";

  if (trimmed.startsWith("{")) {
    try {
      const payload = JSON.parse(trimmed);
      if (payload && typeof payload.Error === "string") return payload.Error;
    } catch {}
  }

  const open = trimmed.indexOf("(");
  const close = trimmed.lastIndexOf(")");
  if (open >= 0 && close > open) {
    try {
      const payload = JSON.parse(trimmed.slice(open + 1, close));
      if (payload && typeof payload.Error === "string") return payload.Error;
    } catch {}
  }

  const xmlError = trimmed.match(/\berror=["']([^"']+)["']/i);
  if (xmlError) return xmlError[1];

  return trimmed.length > 200 ? trimmed.slice(0, 200) : trimmed;
}

function upstreamResponse(env, request, upstream, body, key, attempt) {
  const headers = new Headers();
  copyUpstreamHeaders(headers, upstream.headers);
  addCORS(headers, env, request);
  headers.set("X-OMDB-Manager-Key-Index", String(key.index));
  headers.set("X-OMDB-Manager-Attempts", String(attempt));
  return new Response(request.method === "HEAD" ? null : body, { status: upstream.status, headers });
}

function copyUpstreamHeaders(dst, src) {
  for (const [key, value] of src.entries()) {
    const lower = key.toLowerCase();
    if (HOP_BY_HOP_HEADERS.has(lower)) continue;
    if (lower === "content-length" || lower === "content-encoding") continue;
    dst.append(key, value);
  }
}

function omdbErrorResponse(env, request, status, message) {
  const url = new URL(request.url);
  const responseType = (url.searchParams.get("r") || "").toLowerCase();
  const callback = (url.searchParams.get("callback") || "").trim();
  const headers = new Headers();
  addCORS(headers, env, request);

  if (responseType === "xml") {
    headers.set("Content-Type", "text/xml; charset=utf-8");
    const body = `<?xml version="1.0" encoding="UTF-8"?>\n<root response="False" error="${escapeXML(message)}"></root>`;
    return new Response(request.method === "HEAD" ? null : body, { status, headers });
  }

  const json = JSON.stringify({ Response: "False", Error: message });
  if (callback && JSONP_CALLBACK_RE.test(callback)) {
    headers.set("Content-Type", "application/javascript; charset=utf-8");
    return new Response(request.method === "HEAD" ? null : `${callback}(${json});`, { status, headers });
  }

  headers.set("Content-Type", "application/json; charset=utf-8");
  return new Response(request.method === "HEAD" ? null : json, { status, headers });
}

function jsonResponse(env, request, status, payload) {
  const headers = new Headers({ "Content-Type": "application/json; charset=utf-8" });
  addCORS(headers, env, request);
  return new Response(request.method === "HEAD" ? null : JSON.stringify(payload, null, 2), { status, headers });
}

function docsResponse(env, request) {
  if (request.method !== "GET" && request.method !== "HEAD") {
    return omdbErrorResponse(env, request, 405, "Method not allowed.");
  }
  const headers = new Headers({ "Content-Type": "text/html; charset=utf-8" });
  addCORS(headers, env, request);
  return new Response(request.method === "HEAD" ? null : DOCS_HTML, { status: 200, headers });
}

function optionsResponse(env, request) {
  const headers = new Headers({
    "Allow": "GET, HEAD, POST, OPTIONS",
    "Access-Control-Allow-Methods": "GET, HEAD, POST, OPTIONS",
    "Access-Control-Allow-Headers": "Content-Type, X-API-Key, X-Admin-Key, Authorization"
  });
  addCORS(headers, env, request);
  return new Response(null, { status: 204, headers });
}

function adminStats(request, env, state) {
  if (request.method !== "GET" && request.method !== "HEAD") {
    return jsonResponse(env, request, 405, { error: "method not allowed" });
  }
  if (!adminAuthorized(request, env)) {
    return jsonResponse(env, request, 401, { error: "invalid admin key" });
  }
  return jsonResponse(env, request, 200, {
    clientKeyCount: state.clients.size,
    omdb: state.omdbKeys.stats(true)
  });
}

function adminReload(request, env) {
  if (request.method !== "POST") {
    return jsonResponse(env, request, 405, { error: "method not allowed" });
  }
  if (!adminAuthorized(request, env)) {
    return jsonResponse(env, request, 401, { error: "invalid admin key" });
  }
  const cooldownMs = durationToMs(env.KEY_COOLDOWN || env.KEY_COOLDOWN_MS, DEFAULT_KEY_COOLDOWN_MS);
  runtimeState = new RuntimeState(parseKeys(env.OMDB_KEYS || ""), parseKeys(env.CLIENT_KEYS || ""), cooldownMs, stateSignature(env, cooldownMs));
  return jsonResponse(env, request, 200, {
    ok: true,
    clientKeyCount: runtimeState.clients.size,
    omdb: runtimeState.omdbKeys.stats(false)
  });
}

function adminAuthorized(request, env) {
  if (!env.ADMIN_KEY) return false;
  const url = new URL(request.url);
  const provided = url.searchParams.get("admin_key") || request.headers.get("X-Admin-Key") || "";
  return provided !== "" && provided === env.ADMIN_KEY;
}

function clientAPIKey(request) {
  const url = new URL(request.url);
  const queryKey = (url.searchParams.get("apikey") || "").trim();
  if (queryKey) return queryKey;
  const headerKey = (request.headers.get("X-API-Key") || "").trim();
  if (headerKey) return headerKey;
  const auth = (request.headers.get("Authorization") || "").trim();
  if (auth.toLowerCase().startsWith("bearer ")) return auth.slice(7).trim();
  return "";
}

function normalizePath(pathname) {
  const path = pathname.replace(/\/+$/, "");
  return path || "/";
}

function addCORS(headers, env, request) {
  let origin = env.CORS_ORIGIN || "";
  if (!origin) return;
  if (origin === "reflect") origin = request.headers.get("Origin") || "";
  if (origin) headers.set("Access-Control-Allow-Origin", origin);
}

function durationToMs(value, fallback) {
  if (value === undefined || value === null || String(value).trim() === "") return fallback;
  const raw = String(value).trim();
  if (/^\d+$/.test(raw)) return Number.parseInt(raw, 10);
  const match = raw.match(/^(\d+(?:\.\d+)?)(ms|s|m|h)$/i);
  if (!match) return fallback;
  const amount = Number.parseFloat(match[1]);
  const unit = match[2].toLowerCase();
  const factors = { ms: 1, s: 1000, m: 60_000, h: 3_600_000 };
  return Math.max(0, Math.round(amount * factors[unit]));
}

function maskKey(key) {
  if (!key) return "";
  if (key.length <= 4) return "*".repeat(key.length);
  return `${key.slice(0, 2)}${"*".repeat(Math.max(4, key.length - 4))}${key.slice(-2)}`;
}

function bytesToText(bytes, limit) {
  if (!bytes || bytes.length === 0) return "";
  const sliced = bytes.length > limit ? bytes.slice(0, limit) : bytes;
  return new TextDecoder().decode(sliced);
}

function statusText(status) {
  const map = {
    408: "Request Timeout",
    425: "Too Early",
    429: "Too Many Requests",
    500: "Internal Server Error",
    502: "Bad Gateway",
    503: "Service Unavailable",
    504: "Gateway Timeout"
  };
  return map[status] || "Upstream Error";
}

function escapeXML(value) {
  return String(value)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&apos;");
}
