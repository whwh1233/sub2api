package admin

import (
	"bufio"
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
	CompletedAt           string         `json:"completed_at"`
	RequestID             string         `json:"request_id"`
	ClientRequestID       string         `json:"client_request_id"`
	Method                string         `json:"method"`
	Path                  string         `json:"path"`
	RequestURI            string         `json:"request_uri"`
	RawQuery              string         `json:"raw_query"`
	StatusCode            int            `json:"status_code"`
	LatencyMs             int64          `json:"latency_ms"`
	ClientIP              string         `json:"client_ip"`
	Protocol              string         `json:"protocol"`
	Platform              string         `json:"platform"`
	Model                 string         `json:"model"`
	AccountID             *int64         `json:"account_id,omitempty"`
	RequestBodyBytes      int64          `json:"request_body_bytes"`
	RequestBodyTruncated  bool           `json:"request_body_truncated"`
	ResponseBodyBytes     int64          `json:"response_body_bytes"`
	ResponseBodyTruncated bool           `json:"response_body_truncated"`
	Raw                   map[string]any `json:"raw"`
}

func (h *OpsHandler) ListRawExchangeLogs(c *gin.Context) {
	filter, err := parseRawExchangeLogFilter(c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	path := strings.TrimSpace(os.Getenv("RAW_EXCHANGE_LOG_FILE_PATH"))
	items, total, err := readRawExchangeLogRecords(middleware.RawExchangeLogFilePath(path), filter)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{
		"items": items,
		"total": total,
		"path":  middleware.RawExchangeLogFilePath(path),
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

	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []rawExchangeLogItem{}, 0, nil
		}
		return nil, 0, err
	}
	defer func() { _ = file.Close() }()

	reader := bufio.NewReader(file)
	items := make([]rawExchangeLogItem, 0, filter.Limit)
	total := 0
	var lineNo int64
	for {
		line, readErr := reader.ReadString('\n')
		if line != "" {
			lineNo++
			trimmed := strings.TrimSpace(line)
			if trimmed != "" && rawExchangeLineMatches(trimmed, filter) {
				total++
				var raw map[string]any
				if err := json.Unmarshal([]byte(trimmed), &raw); err != nil {
					return nil, 0, err
				}
				item := rawExchangeLogItemFromRaw(lineNo, raw)
				items = append(items, item)
				if len(items) > filter.Limit {
					copy(items, items[1:])
					items = items[:filter.Limit]
				}
			}
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				break
			}
			return nil, 0, readErr
		}
	}

	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}
	return items, total, nil
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

func rawExchangeLogItemFromRaw(line int64, raw map[string]any) rawExchangeLogItem {
	item := rawExchangeLogItem{
		Line:                  line,
		CompletedAt:           rawString(raw, "completed_at"),
		RequestID:             rawString(raw, "request_id"),
		ClientRequestID:       rawString(raw, "client_request_id"),
		Method:                rawString(raw, "method"),
		Path:                  rawString(raw, "path"),
		RequestURI:            rawString(raw, "request_uri"),
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
