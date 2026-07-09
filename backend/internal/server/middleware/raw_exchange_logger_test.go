package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/gin-gonic/gin"
)

func TestRawExchangeLogger_CapturesOriginalRequestAndResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sink := initMiddlewareTestLogger(t)

	r := gin.New()
	r.Use(RequestLogger())
	r.Use(RawExchangeLogger(config.RawExchangeLogConfig{Enabled: true}))
	r.POST("/v1/chat", func(c *gin.Context) {
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
