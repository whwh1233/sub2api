package rawexchange

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var fileMu sync.Mutex

func FilePath(configured string) string {
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

func AddBody(record map[string]any, prefix string, body []byte) {
	if record == nil {
		return
	}
	copyBody := append([]byte(nil), body...)
	record[prefix] = string(copyBody)
	record[prefix+"_base64"] = base64.StdEncoding.EncodeToString(copyBody)
	record[prefix+"_bytes"] = int64(len(copyBody))
	record[prefix+"_truncated"] = false
}

func Append(configuredPath string, record map[string]any) error {
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	path := FilePath(configuredPath)

	fileMu.Lock()
	defer fileMu.Unlock()

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(data)
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}

func ReadLines(configuredPath string) ([][]byte, error) {
	file, err := os.Open(FilePath(configuredPath))
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	reader := bufio.NewReader(file)
	var lines [][]byte
	for {
		line, readErr := reader.ReadBytes('\n')
		line = []byte(strings.TrimSpace(string(line)))
		if len(line) > 0 {
			lines = append(lines, append([]byte(nil), line...))
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				return lines, nil
			}
			return nil, readErr
		}
	}
}
