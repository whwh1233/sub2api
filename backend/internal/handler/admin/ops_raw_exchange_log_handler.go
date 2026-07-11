package admin

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
)

type rawExchangeLogFilter struct {
	Limit      int
	Query      string
	RequestID  string
	Path       string
	Method     string
	StatusCode int
}

type rawExchangeLogItem struct {
	Line                  int64          `json:"line"`
	Offset                int64          `json:"offset"`
	Stage                 string         `json:"stage"`
	Operation             string         `json:"operation"`
	Attempt               int            `json:"attempt"`
	CompletedAt           string         `json:"completed_at"`
	StartedAt             string         `json:"started_at"`
	RequestID             string         `json:"request_id"`
	ClientRequestID       string         `json:"client_request_id"`
	Method                string         `json:"method"`
	Path                  string         `json:"path"`
	RequestURI            string         `json:"request_uri"`
	URL                   string         `json:"url"`
	RawQuery              string         `json:"raw_query"`
	StatusCode            int            `json:"status_code"`
	LatencyMs             int64          `json:"latency_ms"`
	ClientIP              string         `json:"client_ip"`
	Protocol              string         `json:"protocol"`
	Platform              string         `json:"platform"`
	Model                 string         `json:"model"`
	AccountID             *int64         `json:"account_id,omitempty"`
	UserID                *int64         `json:"user_id,omitempty"`
	RequestBodyBytes      int64          `json:"request_body_bytes"`
	RequestBodyTruncated  bool           `json:"request_body_truncated"`
	ResponseBodyBytes     int64          `json:"response_body_bytes"`
	ResponseBodyTruncated bool           `json:"response_body_truncated"`
	Raw                   map[string]any `json:"raw,omitempty"`
}

func (h *OpsHandler) ListRawExchangeLogs(c *gin.Context) {
	path := strings.TrimSpace(os.Getenv("RAW_EXCHANGE_LOG_FILE_PATH"))
	resolvedPath := middleware.RawExchangeLogFilePath(path)
	if value := strings.TrimSpace(c.Query("offset")); value != "" {
		offset, err := strconv.ParseInt(value, 10, 64)
		if err != nil || offset < 0 {
			response.BadRequest(c, "invalid offset")
			return
		}
		item, err := readRawExchangeLogRecordAt(resolvedPath, offset)
		if err != nil {
			response.Error(c, http.StatusBadRequest, err.Error())
			return
		}
		response.Success(c, item)
		return
	}

	filter, err := parseRawExchangeLogFilter(c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	items, total, err := readRawExchangeLogRecords(resolvedPath, filter)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{
		"items": items,
		"total": total,
		"path":  resolvedPath,
	})
}

func parseRawExchangeLogFilter(c *gin.Context) (rawExchangeLogFilter, error) {
	filter := rawExchangeLogFilter{Limit: 100}
	if c == nil {
		return filter, nil
	}
	if v := strings.TrimSpace(c.Query("limit")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			return filter, errors.New("invalid limit")
		}
		if n > 200 {
			n = 200
		}
		filter.Limit = n
	}
	if v := strings.TrimSpace(c.Query("status_code")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 100 || n > 999 {
			return filter, errors.New("invalid status_code")
		}
		filter.StatusCode = n
	}
	filter.Query = strings.TrimSpace(c.Query("q"))
	filter.RequestID = strings.TrimSpace(c.Query("request_id"))
	filter.Path = strings.TrimSpace(c.Query("path"))
	filter.Method = strings.ToUpper(strings.TrimSpace(c.Query("method")))
	return filter, nil
}

func readRawExchangeLogRecords(path string, filter rawExchangeLogFilter) ([]rawExchangeLogItem, int, error) {
	if filter.Limit <= 0 {
		filter.Limit = 100
	}
	if filter.Limit > 200 {
		filter.Limit = 200
	}

	items := make([]rawExchangeLogItem, 0, filter.Limit)
	err := walkRawExchangeLinesReverse(path, func(offset int64, line []byte) bool {
		trimmed := strings.TrimSpace(string(line))
		if trimmed == "" || !rawExchangeLineMatches(trimmed, filter) {
			return true
		}
		var raw map[string]any
		if json.Unmarshal([]byte(trimmed), &raw) != nil || !rawExchangeRawMatchesClaude(raw) {
			return true
		}
		item := rawExchangeLogItemFromRaw(0, raw)
		item.Offset = offset
		item.Raw = nil
		items = append(items, item)
		return len(items) < filter.Limit
	})
	if os.IsNotExist(err) {
		return []rawExchangeLogItem{}, 0, nil
	}
	return items, len(items), err
}

func walkRawExchangeLinesReverse(path string, visit func(offset int64, line []byte) bool) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	const blockSize int64 = 256 * 1024
	position := info.Size()
	var remainder []byte
	for position > 0 {
		start := position - blockSize
		if start < 0 {
			start = 0
		}
		chunk := make([]byte, position-start)
		if _, err := file.ReadAt(chunk, start); err != nil && !errors.Is(err, io.EOF) {
			return err
		}
		data := append(chunk, remainder...)
		parts := bytes.Split(data, []byte{'\n'})
		offsets := make([]int64, len(parts))
		cursor := start
		for i, part := range parts {
			offsets[i] = cursor
			cursor += int64(len(part)) + 1
		}
		for i := len(parts) - 1; i >= 1; i-- {
			if len(parts[i]) > 0 && !visit(offsets[i], parts[i]) {
				return nil
			}
		}
		remainder = append(remainder[:0], parts[0]...)
		position = start
	}
	if len(remainder) > 0 {
		visit(0, remainder)
	}
	return nil
}

func readRawExchangeLogRecordAt(path string, offset int64) (rawExchangeLogItem, error) {
	file, err := os.Open(path)
	if err != nil {
		return rawExchangeLogItem{}, err
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil || offset < 0 || offset >= info.Size() {
		return rawExchangeLogItem{}, errors.New("invalid log offset")
	}
	if offset > 0 {
		previous := []byte{0}
		if _, err := file.ReadAt(previous, offset-1); err != nil || previous[0] != '\n' {
			return rawExchangeLogItem{}, errors.New("offset is not a log line boundary")
		}
	}
	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return rawExchangeLogItem{}, err
	}
	line, err := bufio.NewReader(file).ReadBytes('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return rawExchangeLogItem{}, err
	}
	var raw map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(line), &raw); err != nil {
		return rawExchangeLogItem{}, err
	}
	item := rawExchangeLogItemFromRaw(0, raw)
	item.Offset = offset
	return item, nil
}

func rawExchangeLineMatches(line string, filter rawExchangeLogFilter) bool {
	lowerLine := strings.ToLower(line)
	if filter.Query != "" && !strings.Contains(lowerLine, strings.ToLower(filter.Query)) {
		return false
	}
	if filter.RequestID != "" && !strings.Contains(line, filter.RequestID) {
		return false
	}
	if filter.Path != "" && !strings.Contains(line, filter.Path) {
		return false
	}
	if filter.Method != "" && !strings.Contains(line, `"method":"`+filter.Method+`"`) {
		return false
	}
	if filter.StatusCode > 0 && !strings.Contains(line, `"status_code":`+strconv.Itoa(filter.StatusCode)) {
		return false
	}
	return true
}

func rawExchangeRawMatchesClaude(raw map[string]any) bool {
	for _, key := range []string{"platform", "model", "path", "request_uri", "request_body"} {
		if rawExchangeContainsClaudeSignal(rawString(raw, key)) {
			return true
		}
	}

	if strings.EqualFold(rawString(raw, "platform"), "anthropic") {
		return true
	}

	headers, _ := raw["request_headers"].(map[string]any)
	for key, value := range headers {
		if strings.EqualFold(key, "Anthropic-Version") || strings.EqualFold(key, "Anthropic-Beta") {
			return true
		}
		if strings.EqualFold(key, "User-Agent") && rawExchangeContainsClaudeSignal(rawExchangeHeaderValueString(value)) {
			return true
		}
	}

	return false
}

func rawExchangeContainsClaudeSignal(value string) bool {
	value = strings.ToLower(value)
	return strings.Contains(value, "claude") || strings.Contains(value, "anthropic")
}

func rawExchangeHeaderValueString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case []string:
		return strings.Join(v, " ")
	case []any:
		parts := make([]string, 0, len(v))
		for _, part := range v {
			if s, ok := part.(string); ok {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, " ")
	default:
		return ""
	}
}

func rawExchangeLogItemFromRaw(line int64, raw map[string]any) rawExchangeLogItem {
	item := rawExchangeLogItem{
		Line:                  line,
		Stage:                 rawString(raw, "stage"),
		Operation:             rawString(raw, "operation"),
		Attempt:               int(rawInt64(raw, "attempt")),
		CompletedAt:           rawString(raw, "completed_at"),
		StartedAt:             rawString(raw, "started_at"),
		RequestID:             rawString(raw, "request_id"),
		ClientRequestID:       rawString(raw, "client_request_id"),
		Method:                rawString(raw, "method"),
		Path:                  rawString(raw, "path"),
		RequestURI:            rawString(raw, "request_uri"),
		URL:                   rawString(raw, "url"),
		RawQuery:              rawString(raw, "raw_query"),
		StatusCode:            int(rawInt64(raw, "status_code")),
		LatencyMs:             rawInt64(raw, "latency_ms"),
		ClientIP:              rawString(raw, "client_ip"),
		Protocol:              rawString(raw, "protocol"),
		Platform:              rawString(raw, "platform"),
		Model:                 rawString(raw, "model"),
		RequestBodyBytes:      rawInt64(raw, "request_body_bytes"),
		RequestBodyTruncated:  rawBool(raw, "request_body_truncated"),
		ResponseBodyBytes:     rawInt64(raw, "response_body_bytes"),
		ResponseBodyTruncated: rawBool(raw, "response_body_truncated"),
		Raw:                   raw,
	}
	if id := rawInt64(raw, "account_id"); id > 0 {
		item.AccountID = &id
	}
	if id := rawInt64(raw, "user_id"); id > 0 {
		item.UserID = &id
	}
	return item
}

func rawString(raw map[string]any, key string) string {
	value, _ := raw[key].(string)
	return value
}

func rawBool(raw map[string]any, key string) bool {
	value, _ := raw[key].(bool)
	return value
}

func rawInt64(raw map[string]any, key string) int64 {
	switch value := raw[key].(type) {
	case int64:
		return value
	case int:
		return int64(value)
	case float64:
		return int64(value)
	case json.Number:
		n, _ := value.Int64()
		return n
	default:
		return 0
	}
}
