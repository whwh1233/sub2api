package rawexchange

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestAppendPreservesExactBodyBytes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit", "raw.jsonl")
	body := []byte{0xff, 'c', 'l', 'a', 'u', 'd', 'e'}
	record := map[string]any{"stage": "upstream_exchange"}
	AddBody(record, "request_body", body)

	if err := Append(path, record); err != nil {
		t.Fatalf("Append() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got["request_body_base64"] != base64.StdEncoding.EncodeToString(body) {
		t.Fatalf("request_body_base64 = %v", got["request_body_base64"])
	}
	if got["request_body_bytes"] != float64(len(body)) {
		t.Fatalf("request_body_bytes = %v", got["request_body_bytes"])
	}
	if got["request_body_truncated"] != false {
		t.Fatalf("request_body_truncated = %v", got["request_body_truncated"])
	}
}

func TestAppendSerializesConcurrentRecords(t *testing.T) {
	path := filepath.Join(t.TempDir(), "raw.jsonl")
	const count = 32
	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			if err := Append(path, map[string]any{"index": index}); err != nil {
				t.Errorf("Append(%d) error = %v", index, err)
			}
		}(i)
	}
	wg.Wait()

	lines, err := ReadLines(path)
	if err != nil {
		t.Fatalf("ReadLines() error = %v", err)
	}
	if len(lines) != count {
		t.Fatalf("line count = %d, want %d", len(lines), count)
	}
	for i, line := range lines {
		var record map[string]any
		if err := json.Unmarshal(line, &record); err != nil {
			t.Fatalf("line %d is not JSON: %v", i, err)
		}
	}
}

func TestFilePathUsesDataDir(t *testing.T) {
	t.Setenv("DATA_DIR", t.TempDir())
	want := filepath.Join(os.Getenv("DATA_DIR"), "raw-exchange", "raw-exchange.jsonl")
	if got := FilePath(""); got != want {
		t.Fatalf("FilePath() = %q, want %q", got, want)
	}
}
