package main

import (
	"context"
	"crypto/subtle"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

//go:embed static/index.html
var docsHTML string

var hopByHopHeaders = map[string]struct{}{
	"connection":          {},
	"keep-alive":          {},
	"proxy-authenticate":  {},
	"proxy-authorization": {},
	"te":                  {},
	"trailer":             {},
	"transfer-encoding":   {},
	"upgrade":             {},
}

type App struct {
	cfg       Config
	client    *http.Client
	dataURL   *url.URL
	posterURL *url.URL
	omdbKeys  *KeyPool
	clients   *AuthStore
}

func NewApp(cfg Config) (*App, error) {
	dataURL, err := url.Parse(cfg.OMDBAPIURL)
	if err != nil {
		return nil, fmt.Errorf("invalid OMDB_API_URL: %w", err)
	}
	posterURL, err := url.Parse(cfg.OMDBPosterURL)
	if err != nil {
		return nil, fmt.Errorf("invalid OMDB_POSTER_URL: %w", err)
	}

	app := &App{
		cfg:       cfg,
		dataURL:   dataURL,
		posterURL: posterURL,
		client: &http.Client{
			Timeout: cfg.HTTPTimeout,
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				DialContext: (&net.Dialer{
					Timeout:   5 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				ForceAttemptHTTP2:     true,
				MaxIdleConns:          100,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			},
		},
	}

	if err := app.ReloadKeys(); err != nil {
		return nil, err
	}
	return app, nil
}

func (a *App) ReloadKeys() error {
	omdbKeys, err := LoadKeys(a.cfg.OMDBKeysFile, os.Getenv("OMDB_KEYS"))
	if err != nil {
		return fmt.Errorf("load OMDb keys: %w", err)
	}
	clientKeys, err := LoadKeys(a.cfg.ClientKeysFile, os.Getenv("CLIENT_KEYS"))
	if err != nil {
		return fmt.Errorf("load client keys: %w", err)
	}

	if a.omdbKeys == nil {
		a.omdbKeys = NewKeyPool(omdbKeys, a.cfg.KeyCooldown)
	} else {
		a.omdbKeys.Reload(omdbKeys)
	}
	if a.clients == nil {
		a.clients = NewAuthStore(clientKeys)
	} else {
		a.clients.Reload(clientKeys)
	}

	log.Printf("loaded %d OMDb key(s) from %s + OMDB_KEYS", len(omdbKeys), a.cfg.OMDBKeysFile)
	log.Printf("loaded %d client key(s) from %s + CLIENT_KEYS", len(clientKeys), a.cfg.ClientKeysFile)
	if len(omdbKeys) == 0 {
		log.Printf("warning: no OMDb upstream keys loaded; API requests will return 503")
	}
	if len(clientKeys) == 0 {
		log.Printf("warning: no client keys loaded; API requests will return 401")
	}
	return nil
}

func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Printf("panic: %v", recovered)
			writeOMDBError(w, r, http.StatusInternalServerError, "Internal server error.")
		}
	}()

	if r.Method == http.MethodOptions {
		a.handleOptions(w, r)
		return
	}

	path := strings.TrimRight(r.URL.Path, "/")
	if path == "" {
		path = "/"
	}

	switch path {
	case "/":
		if r.URL.RawQuery == "" {
			a.handleDocs(w, r)
			return
		}
		a.handleProxy(w, r, a.dataURL)
	case "/api":
		a.handleProxy(w, r, a.dataURL)
	case "/poster":
		a.handleProxy(w, r, a.posterURL)
	case "/docs", "/index.html":
		a.handleDocs(w, r)
	case "/health":
		a.handleHealth(w, r)
	case "/admin/stats":
		a.handleAdminStats(w, r)
	case "/admin/reload":
		a.handleAdminReload(w, r)
	default:
		writeOMDBError(w, r, http.StatusNotFound, "Not found. Use /, /api, /poster, /docs, /health, /admin/stats or /admin/reload.")
	}
}

func (a *App) handleOptions(w http.ResponseWriter, r *http.Request) {
	writeCORS(w, r)
	w.Header().Set("Allow", "GET, HEAD, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key, X-Admin-Key, Authorization")
	w.WriteHeader(http.StatusNoContent)
}

func (a *App) handleDocs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeOMDBError(w, r, http.StatusMethodNotAllowed, "Method not allowed.")
		return
	}
	writeCORS(w, r)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if r.Method != http.MethodHead {
		_, _ = io.WriteString(w, docsHTML)
	}
}

func (a *App) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeOMDBError(w, r, http.StatusMethodNotAllowed, "Method not allowed.")
		return
	}
	omdbStats := a.omdbKeys.Stats(false)
	payload := map[string]any{
		"ok":             omdbStats.TotalKeys > 0 && a.clients.Count() > 0,
		"service":        "omdb-api-manager",
		"clientKeyCount": a.clients.Count(),
		"omdb":           omdbStats,
	}
	writeJSON(w, r, http.StatusOK, payload)
}

func (a *App) handleAdminStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeJSON(w, r, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if !a.adminAuthorized(r) {
		writeJSON(w, r, http.StatusUnauthorized, map[string]string{"error": "invalid admin key"})
		return
	}

	payload := map[string]any{
		"clientKeyCount": a.clients.Count(),
		"omdb":           a.omdbKeys.Stats(true),
	}
	writeJSON(w, r, http.StatusOK, payload)
}

func (a *App) handleAdminReload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, r, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if !a.adminAuthorized(r) {
		writeJSON(w, r, http.StatusUnauthorized, map[string]string{"error": "invalid admin key"})
		return
	}
	if err := a.ReloadKeys(); err != nil {
		writeJSON(w, r, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, r, http.StatusOK, map[string]any{
		"ok":             true,
		"clientKeyCount": a.clients.Count(),
		"omdb":           a.omdbKeys.Stats(false),
	})
}

func (a *App) handleProxy(w http.ResponseWriter, r *http.Request, upstreamBase *url.URL) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeOMDBError(w, r, http.StatusMethodNotAllowed, "Method not allowed. OMDb-compatible requests use GET.")
		return
	}

	if !a.clients.Valid(clientAPIKey(r)) {
		writeOMDBError(w, r, http.StatusUnauthorized, "Invalid API key.")
		return
	}

	if a.omdbKeys.Size() == 0 {
		writeOMDBError(w, r, http.StatusServiceUnavailable, "No upstream OMDb API keys configured.")
		return
	}

	maxAttempts := a.cfg.MaxAttemptsPerRequest
	if maxAttempts <= 0 || maxAttempts > a.omdbKeys.Size() {
		maxAttempts = a.omdbKeys.Size()
	}

	tried := make(map[int]bool, maxAttempts)
	attemptErrors := make([]map[string]any, 0)

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		selected := a.omdbKeys.Acquire(tried)
		if selected == nil {
			break
		}
		tried[selected.Index] = true

		targetURL := buildTargetURL(upstreamBase, r.URL, selected.Value)
		body, resp, err := a.fetchUpstream(r.Context(), r, targetURL)
		if err != nil {
			reason := "network_error"
			if errors.Is(err, context.DeadlineExceeded) || strings.Contains(strings.ToLower(err.Error()), "timeout") {
				reason = "timeout"
			}
			a.omdbKeys.ReportFailure(selected, reason)
			a.omdbKeys.Release(selected)
			attemptErrors = append(attemptErrors, map[string]any{
				"attempt":  attempt,
				"keyIndex": selected.Index,
				"key":      selected.Masked,
				"reason":   reason,
				"message":  err.Error(),
			})
			continue
		}

		contentType := resp.Header.Get("Content-Type")
		retry, reason, message := classifyUpstreamFailure(resp.StatusCode, body, contentType)
		if retry {
			a.omdbKeys.ReportFailure(selected, reason)
			a.omdbKeys.Release(selected)
			attemptErrors = append(attemptErrors, map[string]any{
				"attempt":  attempt,
				"keyIndex": selected.Index,
				"key":      selected.Masked,
				"reason":   reason,
				"message":  message,
			})
			continue
		}

		a.omdbKeys.ReportSuccess(selected)
		a.omdbKeys.Release(selected)
		a.writeUpstreamResponse(w, r, resp, body, selected, attempt)
		return
	}

	if len(attemptErrors) > 0 {
		log.Printf("all attempted OMDb keys failed: %s", mustJSON(attemptErrors))
	}
	writeOMDBError(w, r, http.StatusServiceUnavailable, "All configured OMDb API keys failed or are cooling down.")
}

func (a *App) fetchUpstream(ctx context.Context, original *http.Request, targetURL string) ([]byte, *http.Response, error) {
	ctx, cancel := context.WithTimeout(ctx, a.cfg.HTTPTimeout)
	defer cancel()

	method := http.MethodGet
	req, err := http.NewRequestWithContext(ctx, method, targetURL, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Accept", original.Header.Get("Accept"))
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "*/*")
	}
	if ua := original.Header.Get("User-Agent"); ua != "" {
		req.Header.Set("User-Agent", ua)
	} else {
		req.Header.Set("User-Agent", "omdb-api-manager/1.0")
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp, err
	}
	return body, resp, nil
}

func (a *App) writeUpstreamResponse(w http.ResponseWriter, r *http.Request, resp *http.Response, body []byte, key *ManagedKey, attempt int) {
	writeCORS(w, r)
	copyResponseHeaders(w.Header(), resp.Header)
	w.Header().Set("Content-Length", fmt.Sprint(len(body)))
	w.Header().Set("X-OMDB-Manager-Key-Index", fmt.Sprint(key.Index))
	w.Header().Set("X-OMDB-Manager-Attempts", fmt.Sprint(attempt))
	w.WriteHeader(resp.StatusCode)
	if r.Method != http.MethodHead {
		_, _ = w.Write(body)
	}
}

func buildTargetURL(base *url.URL, incoming *url.URL, upstreamAPIKey string) string {
	target := *base
	target.RawQuery = ""
	query := target.Query()
	for name, values := range incoming.Query() {
		if strings.EqualFold(name, "apikey") {
			continue
		}
		for _, value := range values {
			query.Add(name, value)
		}
	}
	query.Set("apikey", upstreamAPIKey)
	target.RawQuery = query.Encode()
	return target.String()
}

func copyResponseHeaders(dst, src http.Header) {
	for key, values := range src {
		lower := strings.ToLower(key)
		if _, hop := hopByHopHeaders[lower]; hop {
			continue
		}
		if lower == "content-length" || lower == "content-encoding" {
			continue
		}
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

func clientAPIKey(r *http.Request) string {
	if key := strings.TrimSpace(r.URL.Query().Get("apikey")); key != "" {
		return key
	}
	if key := strings.TrimSpace(r.Header.Get("X-API-Key")); key != "" {
		return key
	}
	if auth := strings.TrimSpace(r.Header.Get("Authorization")); strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	return ""
}

func (a *App) adminAuthorized(r *http.Request) bool {
	if a.cfg.AdminKey == "" {
		return false
	}
	provided := strings.TrimSpace(r.URL.Query().Get("admin_key"))
	if provided == "" {
		provided = strings.TrimSpace(r.Header.Get("X-Admin-Key"))
	}
	if provided == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(provided), []byte(a.cfg.AdminKey)) == 1
}

func writeCORS(w http.ResponseWriter, r *http.Request) {
	origin := os.Getenv("CORS_ORIGIN")
	if origin == "" {
		return
	}
	if origin == "reflect" {
		origin = r.Header.Get("Origin")
		if origin == "" {
			return
		}
	}
	w.Header().Set("Access-Control-Allow-Origin", origin)
}

func mustJSON(v any) string {
	body, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprint(v)
	}
	return string(body)
}
