package admin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadRawExchangeLogRecords_ReturnsNewestMatchesWithFullRawEvent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "raw-exchange.jsonl")
	content := `{"request_id":"old","method":"GET","path":"/old","status_code":200,"request_body":"old request","response_body":"old response","completed_at":"2026-07-09T10:00:00Z"}` + "\n" +
		`{"request_id":"new","method":"POST","path":"/latest","raw_query":"token=query-secret","status_code":201,"request_body":"request secret","response_body":"response secret","completed_at":"2026-07-09T10:01:00Z"}` + "\n"
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
		t.Fatalf("total=%d, want 2", total)
	}
	if len(items) != 1 {
		t.Fatalf("items=%d, want 1", len(items))
	}
	item := items[0]
	if item.RequestID != "new" || item.Method != "POST" || item.Path != "/latest" || item.StatusCode != 201 {
		t.Fatalf("summary mismatch: %+v", item)
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
