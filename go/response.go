package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"net/http"
	"regexp"
	"strings"
)

var jsonpCallbackPattern = regexp.MustCompile(`^[A-Za-z_$][0-9A-Za-z_$]*(\.[A-Za-z_$][0-9A-Za-z_$]*)*$`)

func writeJSON(w http.ResponseWriter, r *http.Request, statusCode int, payload any) {
	body, _ := json.MarshalIndent(payload, "", "  ")
	writeCORS(w, r)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Length", fmt.Sprint(len(body)))
	w.WriteHeader(statusCode)
	if r.Method != http.MethodHead {
		_, _ = w.Write(body)
	}
}

func writeOMDBError(w http.ResponseWriter, r *http.Request, statusCode int, message string) {
	query := r.URL.Query()
	responseType := strings.ToLower(strings.TrimSpace(query.Get("r")))
	callback := strings.TrimSpace(query.Get("callback"))

	writeCORS(w, r)

	if responseType == "xml" {
		body := []byte(xml.Header + `<root response="False" error="` + html.EscapeString(message) + `"></root>`)
		w.Header().Set("Content-Type", "text/xml; charset=utf-8")
		w.Header().Set("Content-Length", fmt.Sprint(len(body)))
		w.WriteHeader(statusCode)
		if r.Method != http.MethodHead {
			_, _ = w.Write(body)
		}
		return
	}

	jsonBody, _ := json.Marshal(map[string]string{
		"Response": "False",
		"Error":    message,
	})

	if callback != "" && jsonpCallbackPattern.MatchString(callback) {
		body := []byte(callback + "(" + string(jsonBody) + ");")
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		w.Header().Set("Content-Length", fmt.Sprint(len(body)))
		w.WriteHeader(statusCode)
		if r.Method != http.MethodHead {
			_, _ = w.Write(body)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Length", fmt.Sprint(len(jsonBody)))
	w.WriteHeader(statusCode)
	if r.Method != http.MethodHead {
		_, _ = w.Write(jsonBody)
	}
}

func classifyUpstreamFailure(statusCode int, body []byte, contentType string) (retry bool, reason string, message string) {
	if retryableHTTPStatus(statusCode) {
		return true, fmt.Sprintf("http_%d", statusCode), http.StatusText(statusCode)
	}

	if len(body) == 0 {
		return false, "", ""
	}

	text := body
	if len(text) > 1024*1024 {
		text = text[:1024*1024]
	}
	lower := strings.ToLower(string(text))
	_ = contentType

	switch {
	case strings.Contains(lower, "request limit reached"),
		strings.Contains(lower, "daily limit"),
		strings.Contains(lower, "limit reached"),
		strings.Contains(lower, "quota"),
		strings.Contains(lower, "exceeded"):
		return true, "quota", extractOMDBErrorMessage(body)
	case strings.Contains(lower, "invalid api key"),
		strings.Contains(lower, "invalid apikey"),
		strings.Contains(lower, "no api key"),
		strings.Contains(lower, "api key required"):
		return true, "invalid_key", extractOMDBErrorMessage(body)
	default:
		return false, "", ""
	}
}

func retryableHTTPStatus(statusCode int) bool {
	switch statusCode {
	case http.StatusRequestTimeout,
		425,
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func extractOMDBErrorMessage(body []byte) string {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return ""
	}

	if bytes.HasPrefix(trimmed, []byte("{")) {
		var payload map[string]any
		if json.Unmarshal(trimmed, &payload) == nil {
			if value, ok := payload["Error"].(string); ok {
				return value
			}
		}
	}

	// JSONP: callback({"Response":"False","Error":"..."});
	if idx := bytes.IndexByte(trimmed, '('); idx >= 0 {
		end := bytes.LastIndexByte(trimmed, ')')
		if end > idx {
			var payload map[string]any
			if json.Unmarshal(trimmed[idx+1:end], &payload) == nil {
				if value, ok := payload["Error"].(string); ok {
					return value
				}
			}
		}
	}

	text := string(trimmed)
	if len(text) > 200 {
		return text[:200]
	}
	return text
}
