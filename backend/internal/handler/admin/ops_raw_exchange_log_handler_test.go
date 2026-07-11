package admin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadRawExchangeLogRecords_ReturnsNewestMatchesWithFullRawEvent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "raw-exchange.jsonl")
	content := `{"request_id":"openai","platform":"openai","model":"gpt-test","method":"POST","path":"/v1/chat","status_code":200,"request_body":"openai request secret","response_body":"response secret","completed_at":"2026-07-09T10:00:00Z"}` + "\n" +
		`{"request_id":"old","platform":"anthropic","model":"claude-opus-4-8","method":"POST","path":"/v1/messages","status_code":200,"request_body":"old request secret","response_body":"response secret","completed_at":"2026-07-09T10:01:00Z"}` + "\n" +
		`{"request_id":"new","stage":"upstream_exchange","operation":"messages","attempt":2,"platform":"anthropic","model":"claude-sonnet-4-5","user_id":77,"account_id":9,"method":"POST","url":"https://api.anthropic.com/v1/messages","path":"/v1/messages","raw_query":"token=query-secret","status_code":201,"request_body":"request secret","response_body":"response secret","started_at":"2026-07-09T10:01:59Z","completed_at":"2026-07-09T10:02:00Z"}` + "\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	items, total, err := readRawExchangeLogRecords(path, rawExchangeLogFilter{
		Limit: 1,
		Query: "response",
	})
	if err != nil {
		t.Fatalf("readRawExchangeLogRecords() error: %v", err)
	}
	if total != 1 {
		t.Fatalf("total=%d, want returned summary count", total)
	}
	if len(items) != 1 {
		t.Fatalf("items=%d, want 1", len(items))
	}
	item := items[0]
	if item.RequestID != "new" || item.Method != "POST" || item.Path != "/v1/messages" || item.StatusCode != 201 {
		t.Fatalf("summary mismatch: %+v", item)
	}
	if item.Stage != "upstream_exchange" || item.Operation != "messages" || item.Attempt != 2 || item.UserID == nil || *item.UserID != 77 {
		t.Fatalf("upstream summary mismatch: %+v", item)
	}
	if item.Raw != nil {
		t.Fatalf("list should not return raw payload: %#v", item.Raw)
	}
	detail, err := readRawExchangeLogRecordAt(path, item.Offset)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Raw["request_body"] != "request secret" || detail.Raw["response_body"] != "response secret" {
		t.Fatalf("detail raw=%#v", detail.Raw)
	}
	if detail.Raw["raw_query"] != "token=query-secret" {
		t.Fatalf("raw_query=%v", detail.Raw["raw_query"])
	}
}

func TestReadRawExchangeLogRecordsReturnsSummariesAndOffsets(t *testing.T) {
	path := filepath.Join(t.TempDir(), "raw-exchange.jsonl")
	content := strings.Join([]string{
		`{"request_id":"one","platform":"anthropic","stage":"client_exchange","request_body":"large-secret-one","response_body":"response-one"}`,
		`{"request_id":"two","platform":"anthropic","stage":"upstream_exchange","request_body":"large-secret-two","response_body":"response-two"}`,
	}, "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	items, total, err := readRawExchangeLogRecords(path, rawExchangeLogFilter{Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(items) != 1 || items[0].RequestID != "two" {
		t.Fatalf("items=%+v total=%d", items, total)
	}
	if items[0].Offset <= 0 {
		t.Fatalf("offset=%d", items[0].Offset)
	}
	if items[0].Raw != nil {
		t.Fatalf("list leaked full raw record: %#v", items[0].Raw)
	}

	detail, err := readRawExchangeLogRecordAt(path, items[0].Offset)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Raw["request_body"] != "large-secret-two" || detail.Raw["response_body"] != "response-two" {
		t.Fatalf("detail=%#v", detail.Raw)
	}
}

func TestReadRawExchangeLogRecordAtRejectsInvalidOffset(t *testing.T) {
	path := filepath.Join(t.TempDir(), "raw-exchange.jsonl")
	data, _ := json.Marshal(map[string]any{"platform": "anthropic"})
	if err := os.WriteFile(path, append(data, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readRawExchangeLogRecordAt(path, -1); err == nil {
		t.Fatal("expected invalid offset error")
	}
}
