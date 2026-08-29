package service

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	"github.com/stretchr/testify/require"
)

func TestBuildBillingAttributionText_ClaudeCode241Format(t *testing.T) {
	body := []byte(`{"messages":[{"role":"user","content":"hello from billing"}]}`)
	got, err := buildBillingAttributionText(body, claude.CLICurrentVersion)
	require.NoError(t, err)
	require.Contains(t, got, "x-anthropic-billing-header: cc_version="+claude.CLICurrentVersion+".")
	require.Contains(t, got, "cc_entrypoint=cli;")
	require.Contains(t, got, "cch=00000;")
	fp := computeClaudeCodeFingerprint(body, claude.CLICurrentVersion)
	require.Equal(t,
		"x-anthropic-billing-header: cc_version="+claude.CLICurrentVersion+"."+fp+"; cc_entrypoint=cli; cch=00000;",
		got,
	)
}

func TestComputeClaudeCodeFingerprint_SaltAndIndicesUnchanged(t *testing.T) {
	// 2.1.241 仍使用 salt 59cf53e54c78 与字符下标 4/7/20。
	body := []byte(`{"messages":[{"role":"user","content":"abcdefghijabcdefghijabcdefghij"}]}`)
	got := computeClaudeCodeFingerprint(body, "2.1.241")
	require.Len(t, got, 3)
	require.Equal(t, got, computeClaudeCodeFingerprint(body, "2.1.241"))
}
