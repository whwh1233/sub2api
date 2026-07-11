package xai

import (
	"fmt"
	"net/http"
	"runtime"
	"strings"
)

const (
	// GrokCLIClientVersion must remain at or above the minimum enforced by the
	// Grok CLI inference proxy. This value matches a currently accepted native
	// client version observed in Grok CLI-compatible traffic.
	GrokCLIClientVersion = "0.2.91"

	grokCLIClientIdentifier = "grok-pager"
	grokCLITokenAuth        = "xai-grok-cli"
)

// ApplyGrokCLIHeaders adds the client identity required by
// cli-chat-proxy.grok.com. The proxy reads the version from User-Agent and
// returns HTTP 426 when these headers are absent.
func ApplyGrokCLIHeaders(headers http.Header, model string) {
	if headers == nil {
		return
	}
	userAgent := fmt.Sprintf(
		"grok-pager/%s grok-shell/%s (%s; %s)",
		GrokCLIClientVersion,
		GrokCLIClientVersion,
		runtime.GOOS,
		runtime.GOARCH,
	)
	headers.Set("User-Agent", userAgent)
	headers.Set("x-grok-client-identifier", grokCLIClientIdentifier)
	headers.Set("x-grok-client-version", GrokCLIClientVersion)
	headers.Set("x-xai-token-auth", grokCLITokenAuth)
	if model = strings.TrimSpace(model); model != "" {
		headers.Set("x-grok-model-override", model)
	} else {
		headers.Del("x-grok-model-override")
	}
}
