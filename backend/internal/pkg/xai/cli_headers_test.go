package xai

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApplyGrokCLIHeadersAddsVersionGateIdentity(t *testing.T) {
	headers := make(http.Header)
	ApplyGrokCLIHeaders(headers, "grok-4.5")

	require.Contains(t, headers.Get("User-Agent"), "grok-pager/")
	require.Contains(t, headers.Get("User-Agent"), "grok-shell/")
	require.Equal(t, GrokCLIClientVersion, headers.Get("x-grok-client-version"))
	require.Equal(t, "grok-pager", headers.Get("x-grok-client-identifier"))
	require.Equal(t, "xai-grok-cli", headers.Get("x-xai-token-auth"))
	require.Equal(t, "grok-4.5", headers.Get("x-grok-model-override"))
}

func TestApplyGrokCLIHeadersDoesNotSendEmptyModelOverride(t *testing.T) {
	headers := make(http.Header)
	ApplyGrokCLIHeaders(headers, "  ")

	require.Empty(t, headers.Get("x-grok-model-override"))
}
