package service

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/audit/rawexchange"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"go.uber.org/zap"
)

func (s *GatewayService) doClaudeUpstreamWithTLS(
	req *http.Request,
	proxyURL string,
	account *Account,
	operation string,
	attempt int,
	profile *tlsfingerprint.Profile,
) (*http.Response, error) {
	if s == nil || s.httpUpstream == nil {
		return nil, io.ErrUnexpectedEOF
	}
	if !s.claudeRawExchangeEnabled(account) || req == nil {
		accountID, concurrency := upstreamAccountValues(account)
		return s.httpUpstream.DoWithTLS(req, proxyURL, accountID, concurrency, profile)
	}

	startedAt := time.Now()
	requestBody, requestBodyErr := snapshotHTTPBody(req)
	record := newClaudeUpstreamRecord(req, account, operation, attempt, startedAt)
	rawexchange.AddBody(record, "request_body", requestBody)
	if requestBodyErr != nil {
		record["request_body_read_error"] = requestBodyErr.Error()
	}

	resp, err := s.httpUpstream.DoWithTLS(req, proxyURL, account.ID, account.Concurrency, profile)
	if err != nil {
		record["transport_error"] = err.Error()
		if resp != nil {
			addClaudeResponseMetadata(record, resp)
		}
		completeClaudeUpstreamRecord(s.rawExchangePath(), req, record, nil, startedAt)
		return resp, err
	}
	if resp == nil || resp.Body == nil {
		record["transport_error"] = "upstream returned an empty response body"
		completeClaudeUpstreamRecord(s.rawExchangePath(), req, record, nil, startedAt)
		return resp, nil
	}

	addClaudeResponseMetadata(record, resp)
	resp.Body = &claudeAuditReadCloser{
		ReadCloser: resp.Body,
		finalize: func(body []byte, readErr, closeErr error) {
			if readErr != nil {
				record["response_body_read_error"] = readErr.Error()
			}
			if closeErr != nil {
				record["response_body_close_error"] = closeErr.Error()
			}
			completeClaudeUpstreamRecord(s.rawExchangePath(), req, record, body, startedAt)
		},
	}
	return resp, nil
}

func (s *GatewayService) claudeRawExchangeEnabled(account *Account) bool {
	return s != nil && s.cfg != nil && s.cfg.RawExchangeLog.Enabled && account != nil && account.Platform == PlatformAnthropic
}

func (s *GatewayService) rawExchangePath() string {
	if s == nil || s.cfg == nil {
		return ""
	}
	return s.cfg.RawExchangeLog.FilePath
}

func upstreamAccountValues(account *Account) (int64, int) {
	if account == nil {
		return 0, 0
	}
	return account.ID, account.Concurrency
}

func snapshotHTTPBody(req *http.Request) ([]byte, error) {
	if req == nil || req.Body == nil {
		return nil, nil
	}
	if req.GetBody != nil {
		body, err := req.GetBody()
		if err == nil {
			defer func() { _ = body.Close() }()
			return io.ReadAll(body)
		}
	}
	body, err := io.ReadAll(req.Body)
	_ = req.Body.Close()
	req.Body = io.NopCloser(bytes.NewReader(body))
	return body, err
}

func newClaudeUpstreamRecord(req *http.Request, account *Account, operation string, attempt int, startedAt time.Time) map[string]any {
	ctx := req.Context()
	record := map[string]any{
		"component":       "claude.raw_exchange",
		"stage":           "upstream_exchange",
		"platform":        PlatformAnthropic,
		"operation":       strings.TrimSpace(operation),
		"attempt":         attempt,
		"account_id":      account.ID,
		"account_name":    account.Name,
		"method":          req.Method,
		"url":             req.URL.String(),
		"path":            req.URL.Path,
		"raw_query":       req.URL.RawQuery,
		"request_headers": cloneAuditHeader(req.Header),
		"started_at":      startedAt.Format(time.RFC3339Nano),
	}
	copyContextString(record, "request_id", ctx.Value(ctxkey.RequestID))
	copyContextString(record, "client_request_id", ctx.Value(ctxkey.ClientRequestID))
	copyContextString(record, "model", ctx.Value(ctxkey.Model))
	copyContextInt64(record, "user_id", ctx.Value(ctxkey.UserID))
	return record
}

func addClaudeResponseMetadata(record map[string]any, resp *http.Response) {
	if resp == nil {
		return
	}
	record["status_code"] = resp.StatusCode
	record["response_headers"] = cloneAuditHeader(resp.Header)
}

func completeClaudeUpstreamRecord(path string, req *http.Request, record map[string]any, body []byte, startedAt time.Time) {
	rawexchange.AddBody(record, "response_body", body)
	completedAt := time.Now()
	record["completed_at"] = completedAt.Format(time.RFC3339Nano)
	record["latency_ms"] = completedAt.Sub(startedAt).Milliseconds()
	if err := rawexchange.Append(path, record); err != nil {
		logger.FromContext(req.Context()).Warn("write claude raw exchange jsonl failed", zap.Error(err))
	}
}

func cloneAuditHeader(header http.Header) map[string][]string {
	out := make(map[string][]string, len(header))
	for key, values := range header {
		out[key] = append([]string(nil), values...)
	}
	return out
}

func copyContextString(record map[string]any, key string, value any) {
	if text, ok := value.(string); ok && strings.TrimSpace(text) != "" {
		record[key] = strings.TrimSpace(text)
	}
}

func copyContextInt64(record map[string]any, key string, value any) {
	if number, ok := value.(int64); ok && number > 0 {
		record[key] = number
	}
}

type claudeAuditReadCloser struct {
	io.ReadCloser
	mu       sync.Mutex
	buffer   bytes.Buffer
	readErr  error
	once     sync.Once
	finalize func(body []byte, readErr, closeErr error)
}

func (r *claudeAuditReadCloser) Read(data []byte) (int, error) {
	n, err := r.ReadCloser.Read(data)
	r.mu.Lock()
	if n > 0 {
		_, _ = r.buffer.Write(data[:n])
	}
	if err != nil && err != io.EOF {
		r.readErr = err
	}
	r.mu.Unlock()
	if err == io.EOF {
		r.finish(nil)
	}
	return n, err
}

func (r *claudeAuditReadCloser) Close() error {
	err := r.ReadCloser.Close()
	r.finish(err)
	return err
}

func (r *claudeAuditReadCloser) finish(closeErr error) {
	r.once.Do(func() {
		r.mu.Lock()
		body := append([]byte(nil), r.buffer.Bytes()...)
		readErr := r.readErr
		r.mu.Unlock()
		if r.finalize != nil {
			r.finalize(body, readErr, closeErr)
		}
	})
}
