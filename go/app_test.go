package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeTempFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return path
}

func newTestApp(t *testing.T, upstream http.Handler, omdbKeys string, clientKeys string) *App {
	t.Helper()
	dir := t.TempDir()
	upstreamServer := httptest.NewServer(upstream)
	t.Cleanup(upstreamServer.Close)

	cfg := Config{
		ListenAddr:            ":0",
		OMDBKeysFile:          writeTempFile(t, dir, "omdb_keys.txt", omdbKeys),
		ClientKeysFile:        writeTempFile(t, dir, "client_keys.txt", clientKeys),
		AdminKey:              "admin-secret",
		OMDBAPIURL:            upstreamServer.URL + "/",
		OMDBPosterURL:         upstreamServer.URL + "/poster",
		HTTPTimeout:           2 * time.Second,
		KeyCooldown:           time.Minute,
		MaxAttemptsPerRequest: 0,
	}
	app, err := NewApp(cfg)
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	return app
}

func TestLoadKeys(t *testing.T) {
	dir := t.TempDir()
	path := writeTempFile(t, dir, "keys.txt", "\ufeffa\n# comment\n\nb\na\n")
	keys, err := LoadKeys(path, "c,d c")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"c", "d", "a", "b"}
	if len(keys) != len(want) {
		t.Fatalf("len=%d want %d: %#v", len(keys), len(want), keys)
	}
	for i := range want {
		if keys[i] != want[i] {
			t.Fatalf("keys[%d]=%q want %q; all=%#v", i, keys[i], want[i], keys)
		}
	}
}

func TestProxyRequiresClientKey(t *testing.T) {
	app := newTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("upstream should not be called")
	}), "omdb-good\n", "client-good\n")

	req := httptest.NewRequest(http.MethodGet, "/?t=Inception", nil)
	res := httptest.NewRecorder()
	app.ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
}

func TestProxyReplacesClientAPIKeyAndRetriesQuotaKey(t *testing.T) {
	seenKeys := make([]string, 0)
	app := newTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("apikey")
		seenKeys = append(seenKeys, key)
		w.Header().Set("Content-Type", "application/json")
		if key == "bad" {
			_, _ = w.Write([]byte(`{"Response":"False","Error":"Request limit reached!"}`))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{
			"Response":              "True",
			"Title":                 r.URL.Query().Get("t"),
			"UsedKey":               key,
			"ClientKeyWasForwarded": boolString(key == "client-good"),
		})
	}), "bad\ngood\n", "client-good\n")

	req := httptest.NewRequest(http.MethodGet, "/?apikey=client-good&t=Inception", nil)
	res := httptest.NewRecorder()
	app.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	if len(seenKeys) != 2 || seenKeys[0] != "bad" || seenKeys[1] != "good" {
		t.Fatalf("seenKeys=%#v", seenKeys)
	}

	var payload map[string]string
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json: %v body=%s", err, res.Body.String())
	}
	if payload["UsedKey"] != "good" || payload["Title"] != "Inception" || payload["ClientKeyWasForwarded"] != "false" {
		t.Fatalf("payload=%#v", payload)
	}
}

func TestBusinessErrorDoesNotRetry(t *testing.T) {
	seen := 0
	app := newTestApp(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"Response":"False","Error":"Movie not found!"}`))
	}), "k1\nk2\n", "client-good\n")

	req := httptest.NewRequest(http.MethodGet, "/?apikey=client-good&t=NoSuchMovie", nil)
	res := httptest.NewRecorder()
	app.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	if seen != 1 {
		t.Fatalf("seen=%d want 1", seen)
	}
}

func boolString(v bool) string {
	if v {
		return "true"
	}
	return "false"
}
