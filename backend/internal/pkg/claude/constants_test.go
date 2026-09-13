package claude

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Preserve the custom CLI headers and cache betas while accepting the upstream Fable 5.1 baseline.
func TestCLIIdentityPreservesCustomHeadersAndUpstreamBaseline(t *testing.T) {
	require.Equal(t, "2.1.258", CLICurrentVersion)
	require.Equal(t, "0.112.1", CLIStainlessPackageVersion)
	require.Equal(t, "claude_code_cli", CLIClientPlatform)
	require.Equal(t, "claude-cli/2.1.258 (external, cli)", DefaultHeaders["User-Agent"])
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
		BetaThinkingBindingControls,
		BetaExtendedCacheTTL,
	}, FullClaudeCodeMimicryBetas())
	require.NotContains(t, FullClaudeCodeMimicryBetas(), BetaRedactThinking)
	require.NotContains(t, FullClaudeCodeMimicryBetas(), BetaContext1M)
	require.NotContains(t, FullClaudeCodeMimicryBetas(), BetaFastMode)
}
