package middleware

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/gin-gonic/gin"
)

func TestRawExchangeLogger_CapturesOriginalRequestAndResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sink := initMiddlewareTestLogger(t)

	r := gin.New()
	r.Use(RequestLogger())
	r.Use(RawExchangeLogger(config.RawExchangeLogConfig{
		Enabled:  true,
		FilePath: filepath.Join(t.TempDir(), "raw-exchange.jsonl"),
	}))
	r.POST("/v1/chat", func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), ctxkey.Platform, "anthropic")
		ctx = context.WithValue(ctx, ctxkey.UserID, int64(77))
		c.Request = c.Request.WithContext(ctx)
		c.Header("X-Upstream-Token", "response-header-secret")
		c.JSON(http.StatusAccepted, gin.H{
			"message": "ok",
			"token":   "response-body-secret",
		})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/v1/chat?api_key=query-secret&model=gpt-test",
		strings.NewReader(`{"password":"request-body-secret","prompt":"hello"}`),
	)
	req.Header.Set("Authorization", "Bearer request-header-secret")
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("status=%d", w.Code)
	}

	event := findRawExchangeEvent(t, sink.list())
	assertStringField(t, event, "request_body", `"password":"request-body-secret"`)
	assertStringField(t, event, "response_body", `"token":"response-body-secret"`)
	assertStringField(t, event, "raw_query", "api_key=query-secret")
	assertStringField(t, event, "stage", "client_exchange")
	if got := event.Fields["user_id"]; got != int64(77) {
		t.Fatalf("user_id=%v", got)
	}

	requestHeaders, ok := event.Fields["request_headers"].(map[string][]string)
	if !ok {
		t.Fatalf("request_headers type=%T, want map[string][]string", event.Fields["request_headers"])
	}
	if got := strings.Join(requestHeaders["Authorization"], ","); got != "Bearer request-header-secret" {
		t.Fatalf("Authorization header=%q", got)
	}

	responseHeaders, ok := event.Fields["response_headers"].(map[string][]string)
	if !ok {
		t.Fatalf("response_headers type=%T, want map[string][]string", event.Fields["response_headers"])
	}
	if got := strings.Join(responseHeaders["X-Upstream-Token"], ","); got != "response-header-secret" {
		t.Fatalf("X-Upstream-Token=%q", got)
	}
}

func TestRawExchangeLogger_SkipsNonClaudeRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sink := initMiddlewareTestLogger(t)
	logPath := filepath.Join(t.TempDir(), "raw-exchange.jsonl")

	r := gin.New()
	r.Use(RequestLogger())
	r.Use(RawExchangeLogger(config.RawExchangeLogConfig{
		Enabled:  true,
		FilePath: logPath,
	}))
	r.POST("/v1/chat", func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), ctxkey.Platform, "openai")
		ctx = context.WithValue(ctx, ctxkey.Model, "gpt-test")
		c.Request = c.Request.WithContext(ctx)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/chat", strings.NewReader(`{"model":"gpt-test"}`))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	if hasRawExchangeEvent(sink.list()) {
		t.Fatalf("raw exchange should not be logged for non-Claude requests")
	}
	if _, err := os.Stat(logPath); err == nil {
		t.Fatalf("raw exchange file should not be created for non-Claude requests")
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat raw exchange file: %v", err)
	}
}

func TestRawExchangeLogger_LogsAnthropicHeaderRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sink := initMiddlewareTestLogger(t)

	r := gin.New()
	r.Use(RequestLogger())
	r.Use(RawExchangeLogger(config.RawExchangeLogConfig{
		Enabled:  true,
		FilePath: filepath.Join(t.TempDir(), "raw-exchange.jsonl"),
	}))
	r.POST("/v1/messages", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"id": "msg_1"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"model":"claude-sonnet-4-5"}`))
	req.Header.Set("anthropic-version", "2023-06-01")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	event := findRawExchangeEvent(t, sink.list())
	assertStringField(t, event, "request_body", "claude-sonnet-4-5")
	assertStringField(t, event, "path", "/v1/messages")
}

func TestRawExchangeLogger_DisabledDoesNotLog(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sink := initMiddlewareTestLogger(t)

	r := gin.New()
	r.Use(RequestLogger())
	r.Use(RawExchangeLogger(config.RawExchangeLogConfig{Enabled: false}))
	r.POST("/v1/chat", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/chat", strings.NewReader("secret"))
	r.ServeHTTP(w, req)

	for _, event := range sink.list() {
		if event != nil && event.Message == "http raw exchange captured" {
			t.Fatalf("raw exchange should not be logged when disabled: %+v", event.Fields)
		}
	}
}

func TestRawExchangeLogger_WritesRawJSONLFileWithExactBodyBytes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	_ = initMiddlewareTestLogger(t)
	dataDir := t.TempDir()
	t.Setenv("DATA_DIR", dataDir)

	requestBody := []byte{0xff, 'r', 'a', 'w'}
	responseBody := []byte{0xfe, 'o', 'k'}

	r := gin.New()
	r.Use(RequestLogger())
	r.Use(RawExchangeLogger(config.RawExchangeLogConfig{Enabled: true}))
	r.POST("/v1/raw", func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), ctxkey.Platform, "anthropic")
		c.Request = c.Request.WithContext(ctx)
		c.Header("X-Raw-Response", "response-secret")
		c.Data(http.StatusCreated, "application/octet-stream", responseBody)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/raw?token=query-secret", bytes.NewReader(requestBody))
	req.Header.Set("Authorization", "Bearer request-secret")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status=%d", w.Code)
	}

	logPath := filepath.Join(dataDir, "raw-exchange", "raw-exchange.jsonl")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read raw exchange log file: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 1 {
		t.Fatalf("log lines=%d, want 1: %q", len(lines), string(data))
	}

	var record map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &record); err != nil {
		t.Fatalf("unmarshal raw exchange jsonl: %v\nline=%s", err, lines[0])
	}
	if got := record["request_body_base64"]; got != base64.StdEncoding.EncodeToString(requestBody) {
		t.Fatalf("request_body_base64=%v", got)
	}
	if got := record["response_body_base64"]; got != base64.StdEncoding.EncodeToString(responseBody) {
		t.Fatalf("response_body_base64=%v", got)
	}
	if got := record["raw_query"]; got != "token=query-secret" {
		t.Fatalf("raw_query=%v", got)
	}
	headers, ok := record["request_headers"].(map[string]any)
	if !ok {
		t.Fatalf("request_headers type=%T", record["request_headers"])
	}
	if got := headers["Authorization"]; got == nil {
		t.Fatalf("Authorization header missing in raw JSONL record")
	}
}

func findRawExchangeEvent(t *testing.T, events []*logger.LogEvent) *logger.LogEvent {
	t.Helper()
	for _, event := range events {
		if event != nil && event.Message == "http raw exchange captured" {
			return event
		}
	}
	t.Fatalf("raw exchange log event not found in %d events", len(events))
	return nil
}

func hasRawExchangeEvent(events []*logger.LogEvent) bool {
	for _, event := range events {
		if event != nil && event.Message == "http raw exchange captured" {
			return true
		}
	}
	return false
}

func assertStringField(t *testing.T, event *logger.LogEvent, field string, wantContains string) {
	t.Helper()
	got, ok := event.Fields[field].(string)
	if !ok {
		t.Fatalf("%s type=%T, want string", field, event.Fields[field])
	}
	if !strings.Contains(got, wantContains) {
		t.Fatalf("%s=%q, want containing %q", field, got, wantContains)
	}
}
