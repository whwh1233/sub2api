package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
)

type claudeAuditUpstreamStub struct {
	response *http.Response
	err      error
}

func (s *claudeAuditUpstreamStub) Do(req *http.Request, proxyURL string, accountID int64, accountConcurrency int) (*http.Response, error) {
	return s.response, s.err
}

func (s *claudeAuditUpstreamStub) DoWithTLS(req *http.Request, proxyURL string, accountID int64, accountConcurrency int, profile *tlsfingerprint.Profile) (*http.Response, error) {
	return s.response, s.err
}

func TestDoClaudeUpstreamWithTLSCapturesExactExchange(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "raw.jsonl")
	responseBody := []byte{0xfe, 's', 's', 'e'}
	upstream := &claudeAuditUpstreamStub{response: &http.Response{
		StatusCode: http.StatusCreated,
		Header:     http.Header{"X-Upstream-Secret": []string{"response-secret"}},
		Body:       io.NopCloser(bytes.NewReader(responseBody)),
	}}
	svc := &GatewayService{
		cfg:          &config.Config{RawExchangeLog: config.RawExchangeLogConfig{Enabled: true, FilePath: logPath}},
		httpUpstream: upstream,
	}
	requestBody := []byte{0xff, 'p', 'r', 'o', 'm', 'p', 't'}
	req, err := http.NewRequest(http.MethodPost, "https://api.anthropic.com/v1/messages?beta=true", bytes.NewReader(requestBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer upstream-secret")
	ctx := context.WithValue(context.Background(), ctxkey.RequestID, "request-123")
	ctx = context.WithValue(ctx, ctxkey.ClientRequestID, "client-456")
	ctx = context.WithValue(ctx, ctxkey.UserID, int64(77))
	ctx = context.WithValue(ctx, ctxkey.Model, "claude-sonnet")
	req = req.WithContext(ctx)

	resp, err := svc.doClaudeUpstreamWithTLS(req, "", &Account{ID: 9, Platform: PlatformAnthropic, Concurrency: 1}, "messages", 2, nil)
	if err != nil {
		t.Fatalf("doClaudeUpstreamWithTLS() error = %v", err)
	}
	gotBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if err := resp.Body.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if !bytes.Equal(gotBody, responseBody) {
		t.Fatalf("response body changed: %x", gotBody)
	}

	record := readSingleClaudeAuditRecord(t, logPath)
	assertClaudeAuditField(t, record, "stage", "upstream_exchange")
	assertClaudeAuditField(t, record, "request_id", "request-123")
	assertClaudeAuditField(t, record, "client_request_id", "client-456")
	assertClaudeAuditField(t, record, "operation", "messages")
	if record["attempt"] != float64(2) || record["user_id"] != float64(77) || record["account_id"] != float64(9) {
		t.Fatalf("identity fields = %#v", record)
	}
	if record["request_body_bytes"] != float64(len(requestBody)) || record["response_body_bytes"] != float64(len(responseBody)) {
		t.Fatalf("body sizes = %#v", record)
	}
	requestHeaders := record["request_headers"].(map[string]any)
	if requestHeaders["Authorization"].([]any)[0] != "Bearer upstream-secret" {
		t.Fatalf("request headers = %#v", requestHeaders)
	}
}

func TestDoClaudeUpstreamWithTLSCapturesTransportError(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "raw.jsonl")
	svc := &GatewayService{
		cfg:          &config.Config{RawExchangeLog: config.RawExchangeLogConfig{Enabled: true, FilePath: logPath}},
		httpUpstream: &claudeAuditUpstreamStub{err: errors.New("dial secret.example: timeout")},
	}
	req, _ := http.NewRequest(http.MethodPost, "https://api.anthropic.com/v1/messages", bytes.NewReader([]byte(`{"model":"claude"}`)))

	_, err := svc.doClaudeUpstreamWithTLS(req, "", &Account{ID: 9, Platform: PlatformAnthropic}, "messages", 1, nil)
	if err == nil {
		t.Fatal("expected transport error")
	}
	record := readSingleClaudeAuditRecord(t, logPath)
	assertClaudeAuditField(t, record, "transport_error", "dial secret.example: timeout")
}

func TestDoClaudeUpstreamWithTLSSkipsNonAnthropic(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "raw.jsonl")
	svc := &GatewayService{
		cfg: &config.Config{RawExchangeLog: config.RawExchangeLogConfig{Enabled: true, FilePath: logPath}},
		httpUpstream: &claudeAuditUpstreamStub{response: &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(bytes.NewReader([]byte("ok"))),
		}},
	}
	req, _ := http.NewRequest(http.MethodPost, "https://api.openai.com/v1/responses", bytes.NewReader([]byte(`{"model":"gpt"}`)))
	resp, err := svc.doClaudeUpstreamWithTLS(req, "", &Account{ID: 10, Platform: PlatformOpenAI}, "responses", 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Fatalf("non-Anthropic request created log: %v", err)
	}
}

func readSingleClaudeAuditRecord(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var record map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(data), &record); err != nil {
		t.Fatalf("invalid JSONL: %v\n%s", err, data)
	}
	return record
}

func assertClaudeAuditField(t *testing.T, record map[string]any, key, want string) {
	t.Helper()
	if got := record[key]; got != want {
		t.Fatalf("%s = %v, want %q", key, got, want)
	}
}
