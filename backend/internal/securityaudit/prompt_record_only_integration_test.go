package securityaudit

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"strings"
	"testing"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

const promptAuditRecordOnlyPostgresTestEnv = "PROMPT_AUDIT_RECORD_ONLY_TEST_POSTGRES_DSN"

func TestRecordOnlyPersistsFullPromptInPostgres(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv(promptAuditRecordOnlyPostgresTestEnv))
	if dsn == "" {
		t.Skip(promptAuditRecordOnlyPostgresTestEnv + " is not set")
	}
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, db.Ping())

	const canary = "RECORD_ONLY_POSTGRES_CANARY_20260827"
	snapshot, err := ExtractPromptSnapshot(Request{
		RequestID: "record-only-postgres-integration",
		Protocol:  "anthropic_messages",
		Model:     "claude-test",
		Body:      []byte(`{"messages":[{"role":"user","content":"` + canary + `"}]}`),
	})
	require.NoError(t, err)

	repo := NewPostgreSQLRepository(db)
	job, err := repo.CreateStagingWithCapacity(context.Background(), snapshot.Redacted(), 1, 3, 100000)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM prompt_audit_events WHERE job_id=$1`, job.ID)
		_, _ = db.Exec(`DELETE FROM prompt_audit_jobs WHERE id=$1`, job.ID)
	})

	require.NoError(t, db.QueryRow(`
		UPDATE prompt_audit_jobs
		SET status='processing', attempts=1, claim_version=claim_version+1, processing_started_at=NOW()
		WHERE id=$1
		RETURNING claim_version`, job.ID).Scan(&job.ClaimVersion))
	job.Attempts = 1
	job.MaxAttempts = 3
	job.ConfigVersion = 1

	cfg := ActiveConfig{Enabled: true, RecordOnly: true, WorkerCount: 1, QueueCapacity: 100000, AllGroups: true, ConfigVersion: 1}
	payload := &fakePayloadStore{values: map[int64]string{job.ID: snapshot.ScanText}}
	scannerCalls := 0
	runner := NewRunner(&fakeConfigStore{cfg: cfg, active: true}, repo, payload, PromptScannerFunc(func(context.Context, ActiveEndpoint, string, []string) (*NormalizedResult, error) {
		scannerCalls++
		return nil, errors.New("record-only must not call scanner")
	}), NewAtomicMetrics())
	require.NoError(t, runner.processJob(context.Background(), 0, cfg, job))
	require.Zero(t, scannerCalls)

	var fullPrompt, scannerBackend string
	require.NoError(t, db.QueryRow(`SELECT full_prompt,scanner_backend FROM prompt_audit_events WHERE job_id=$1`, job.ID).Scan(&fullPrompt, &scannerBackend))
	require.Contains(t, fullPrompt, canary)
	require.Equal(t, "record-only", scannerBackend)
}
