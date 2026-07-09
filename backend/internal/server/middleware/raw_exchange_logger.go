package middleware

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var rawExchangeFileMu sync.Mutex

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
		record := map[string]any{
			"component":               "http.raw_exchange",
			"request_id":              strings.TrimSpace(requestID),
			"client_request_id":       strings.TrimSpace(clientRequestID),
			"method":                  c.Request.Method,
			"path":                    c.Request.URL.Path,
			"request_uri":             c.Request.RequestURI,
			"raw_query":               c.Request.URL.RawQuery,
			"query":                   cloneValues(c.Request.URL.Query()),
			"path_params":             cloneParams(c.Params),
			"request_headers":         cloneHeader(c.Request.Header),
			"request_body":            capturedRequest.value,
			"request_body_base64":     base64.StdEncoding.EncodeToString(capturedRequest.bytes),
			"request_body_bytes":      int64(len(requestBody)),
			"request_body_truncated":  capturedRequest.truncated,
			"status_code":             c.Writer.Status(),
			"response_headers":        cloneHeader(c.Writer.Header()),
			"response_body":           responseBody.String(),
			"response_body_base64":    base64.StdEncoding.EncodeToString(responseBody.Bytes()),
			"response_body_bytes":     responseBody.TotalBytes(),
			"response_body_truncated": responseBody.Truncated(),
			"latency_ms":              endTime.Sub(startTime).Milliseconds(),
			"client_ip":               ip.GetClientIP(c),
			"protocol":                c.Request.Proto,
			"completed_at":            endTime.Format(time.RFC3339Nano),
		}
		if requestReadErr != nil {
			record["request_body_read_error"] = requestReadErr.Error()
		}
		if hasAccountID && accountID > 0 {
			record["account_id"] = accountID
		}
		if platform != "" {
			record["platform"] = platform
		}
		if model != "" {
			record["model"] = model
		}

		fields := rawExchangeZapFields(record)
		logger.FromContext(ctx).With(fields...).Info("http raw exchange captured")
		if err := appendRawExchangeJSONL(cfg.FilePath, record); err != nil {
			logger.FromContext(ctx).Warn("write raw exchange jsonl failed", zap.Error(err))
		}
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
	bytes     []byte
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
		return capturedText{value: string(data), bytes: append([]byte(nil), data...)}
	}
	return capturedText{
		value:     string(data[:maxBytes]),
		bytes:     append([]byte(nil), data[:maxBytes]...),
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

func (b *captureBuffer) Bytes() []byte {
	return append([]byte(nil), b.buffer.Bytes()...)
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

func rawExchangeZapFields(record map[string]any) []zap.Field {
	fields := make([]zap.Field, 0, len(record))
	for key, value := range record {
		switch v := value.(type) {
		case string:
			fields = append(fields, zap.String(key, v))
		case int:
			fields = append(fields, zap.Int(key, v))
		case int64:
			fields = append(fields, zap.Int64(key, v))
		case bool:
			fields = append(fields, zap.Bool(key, v))
		default:
			fields = append(fields, zap.Any(key, v))
		}
	}
	return fields
}

// RawExchangeLogFilePath returns the JSONL file used by the test-only raw
// exchange viewer. Empty config resolves under DATA_DIR so systemd deployments
// keep the file with the rest of the app data.
func RawExchangeLogFilePath(configured string) string {
	configured = strings.TrimSpace(configured)
	if configured != "" {
		return configured
	}
	dataDir := strings.TrimSpace(os.Getenv("DATA_DIR"))
	if dataDir == "" {
		dataDir = "."
	}
	return filepath.Join(dataDir, "raw-exchange", "raw-exchange.jsonl")
}

func appendRawExchangeJSONL(configuredPath string, record map[string]any) error {
	path := RawExchangeLogFilePath(configuredPath)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(record); err != nil {
		return err
	}

	rawExchangeFileMu.Lock()
	defer rawExchangeFileMu.Unlock()

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	_, err = file.Write(buf.Bytes())
	return err
}
