package claude

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCLIIdentityMatchesClaudeCode241 把伪装身份钉在本机官方 CLI 2.1.241 的提取结果上。
// 升级 Claude Code 后如果这些值变了，应先对照二进制再改常量，避免 UA / beta / stainless 漂移。
func TestCLIIdentityMatchesClaudeCode241(t *testing.T) {
	require.Equal(t, "2.1.241", CLICurrentVersion)
	require.Equal(t, "0.112.1", CLIStainlessPackageVersion)
	require.Equal(t, "claude_code_cli", CLIClientPlatform)
	require.Equal(t, "claude-cli/2.1.241 (external, cli)", DefaultHeaders["User-Agent"])
	require.Equal(t, CLIStainlessPackageVersion, DefaultHeaders["X-Stainless-Package-Version"])
	require.Equal(t, CLIClientPlatform, DefaultHeaders["anthropic-client-platform"])
	require.Equal(t, "cli", DefaultHeaders["X-App"])
	require.Equal(t, []string{
		BetaClaudeCode,
		BetaOAuth,
		BetaInterleavedThinking,
		BetaContextManagement,
		BetaEffort,
		BetaPromptCachingScope,
		BetaPromptCachingEvict,
		BetaExtendedCacheTTL,
	}, FullClaudeCodeMimicryBetas())
	require.NotContains(t, FullClaudeCodeMimicryBetas(), BetaRedactThinking)
	require.NotContains(t, FullClaudeCodeMimicryBetas(), BetaContext1M)
	require.NotContains(t, FullClaudeCodeMimicryBetas(), BetaFastMode)
}
