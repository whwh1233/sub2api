package admin

import (
	"os"
	"path/filepath"
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
	if total != 2 {
		t.Fatalf("total=%d, want 2 Claude records only", total)
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
	if item.Raw["request_body"] != "request secret" {
		t.Fatalf("raw request_body=%v", item.Raw["request_body"])
	}
	if item.Raw["response_body"] != "response secret" {
		t.Fatalf("raw response_body=%v", item.Raw["response_body"])
	}
	if item.Raw["raw_query"] != "token=query-secret" {
		t.Fatalf("raw_query=%v", item.Raw["raw_query"])
	}
}
