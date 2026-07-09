package middleware

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RawExchangeLogger captures raw request and response data for debugging
// deployments. It intentionally does not redact anything.
func RawExchangeLogger(cfg config.RawExchangeLogConfig) gin.HandlerFunc {
	skipPaths := make(map[string]struct{}, len(cfg.SkipPaths))
	for _, path := range cfg.SkipPaths {
		path = strings.TrimSpace(path)
		if path != "" {
			skipPaths[path] = struct{}{}
		}
	}

	return func(c *gin.Context) {
		if !cfg.Enabled || c.Request == nil {
			c.Next()
			return
		}
		if _, skip := skipPaths[c.Request.URL.Path]; skip {
			c.Next()
			return
		}

		startTime := time.Now()
		requestBody, requestReadErr := readAndRestoreRequestBody(c.Request)

		responseBody := newCaptureBuffer(cfg.MaxBodyBytes)
		originalWriter := c.Writer
		c.Writer = &rawExchangeResponseWriter{
			ResponseWriter: originalWriter,
			body:           responseBody,
		}

		c.Next()

		endTime := time.Now()
		ctx := c.Request.Context()
		requestID, _ := ctx.Value(ctxkey.RequestID).(string)
		clientRequestID, _ := ctx.Value(ctxkey.ClientRequestID).(string)
		accountID, hasAccountID := ctx.Value(ctxkey.AccountID).(int64)
		platform, _ := ctx.Value(ctxkey.Platform).(string)
		model, _ := ctx.Value(ctxkey.Model).(string)

		capturedRequest := captureBytes(requestBody, cfg.MaxBodyBytes)
		fields := []zap.Field{
			zap.String("component", "http.raw_exchange"),
			zap.String("request_id", strings.TrimSpace(requestID)),
			zap.String("client_request_id", strings.TrimSpace(clientRequestID)),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("request_uri", c.Request.RequestURI),
			zap.String("raw_query", c.Request.URL.RawQuery),
			zap.Any("query", cloneValues(c.Request.URL.Query())),
			zap.Any("path_params", cloneParams(c.Params)),
			zap.Any("request_headers", cloneHeader(c.Request.Header)),
			zap.String("request_body", capturedRequest.value),
			zap.Int64("request_body_bytes", int64(len(requestBody))),
			zap.Bool("request_body_truncated", capturedRequest.truncated),
			zap.Int("status_code", c.Writer.Status()),
			zap.Any("response_headers", cloneHeader(c.Writer.Header())),
			zap.String("response_body", responseBody.String()),
			zap.Int64("response_body_bytes", responseBody.TotalBytes()),
			zap.Bool("response_body_truncated", responseBody.Truncated()),
			zap.Int64("latency_ms", endTime.Sub(startTime).Milliseconds()),
			zap.String("client_ip", ip.GetClientIP(c)),
			zap.String("protocol", c.Request.Proto),
			zap.Time("completed_at", endTime),
		}
		if requestReadErr != nil {
			fields = append(fields, zap.String("request_body_read_error", requestReadErr.Error()))
		}
		if hasAccountID && accountID > 0 {
			fields = append(fields, zap.Int64("account_id", accountID))
		}
		if platform != "" {
			fields = append(fields, zap.String("platform", platform))
		}
		if model != "" {
			fields = append(fields, zap.String("model", model))
		}

		logger.FromContext(ctx).With(fields...).Info("http raw exchange captured")
	}
}

type rawExchangeResponseWriter struct {
	gin.ResponseWriter
	body *captureBuffer
}

func (w *rawExchangeResponseWriter) Write(data []byte) (int, error) {
	if w.body != nil {
		_, _ = w.body.Write(data)
	}
	return w.ResponseWriter.Write(data)
}

func (w *rawExchangeResponseWriter) WriteString(data string) (int, error) {
	if w.body != nil {
		_, _ = w.body.WriteString(data)
	}
	return w.ResponseWriter.WriteString(data)
}

type capturedText struct {
	value     string
	truncated bool
}

func readAndRestoreRequestBody(req *http.Request) ([]byte, error) {
	if req == nil || req.Body == nil {
		return nil, nil
	}
	body, err := io.ReadAll(req.Body)
	_ = req.Body.Close()
	req.Body = io.NopCloser(bytes.NewReader(body))
	return body, err
}

func captureBytes(data []byte, maxBytes int64) capturedText {
	if maxBytes <= 0 || int64(len(data)) <= maxBytes {
		return capturedText{value: string(data)}
	}
	return capturedText{
		value:     string(data[:maxBytes]),
		truncated: true,
	}
}

type captureBuffer struct {
	buffer    bytes.Buffer
	maxBytes  int64
	total     int64
	truncated bool
}

func newCaptureBuffer(maxBytes int64) *captureBuffer {
	return &captureBuffer{maxBytes: maxBytes}
}

func (b *captureBuffer) Write(data []byte) (int, error) {
	b.total += int64(len(data))
	if b.maxBytes <= 0 {
		_, _ = b.buffer.Write(data)
		return len(data), nil
	}
	remaining := b.maxBytes - int64(b.buffer.Len())
	if remaining <= 0 {
		if len(data) > 0 {
			b.truncated = true
		}
		return len(data), nil
	}
	if int64(len(data)) > remaining {
		_, _ = b.buffer.Write(data[:remaining])
		b.truncated = true
		return len(data), nil
	}
	_, _ = b.buffer.Write(data)
	return len(data), nil
}

func (b *captureBuffer) WriteString(data string) (int, error) {
	return b.Write([]byte(data))
}

func (b *captureBuffer) String() string {
	return b.buffer.String()
}

func (b *captureBuffer) TotalBytes() int64 {
	return b.total
}

func (b *captureBuffer) Truncated() bool {
	return b.truncated
}

func cloneHeader(header http.Header) map[string][]string {
	out := make(map[string][]string, len(header))
	for key, values := range header {
		out[key] = append([]string(nil), values...)
	}
	return out
}

func cloneValues(values map[string][]string) map[string][]string {
	out := make(map[string][]string, len(values))
	for key, value := range values {
		out[key] = append([]string(nil), value...)
	}
	return out
}

func cloneParams(params gin.Params) map[string]string {
	out := make(map[string]string, len(params))
	for _, param := range params {
		out[param.Key] = param.Value
	}
	return out
}
