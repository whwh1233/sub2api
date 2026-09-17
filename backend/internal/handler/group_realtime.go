package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// Count inference operations, not metadata, token counting, or async polling.
func isGroupRealtimeRequest(c *gin.Context) bool {
	path := strings.TrimRight(c.Request.URL.Path, "/")
	if c.Request.Method == http.MethodGet {
		return strings.EqualFold(c.GetHeader("Upgrade"), "websocket") && strings.HasSuffix(path, "/responses")
	}
	if c.Request.Method != http.MethodPost {
		return false
	}
	for _, suffix := range []string{"/messages", "/responses", "/responses/compact", "/chat/completions", "/completions", "/images/generations", "/images/edits", "/images/generations/async", "/images/edits/async", "/images/batches", "/videos", "/videos/generations", "/videos/edits", "/videos/extensions", "/audio/speech", "/audio/transcriptions", "/audio/translations", "/tts", "/stt", "/alpha/search", "/web_search", "/x_search", "/embeddings", ":generateContent", ":streamGenerateContent"} {
		if strings.HasSuffix(path, suffix) {
			return true
		}
	}
	return false
}

func groupRealtimeOutcome(c *gin.Context, w *opsCaptureWriter, panicked bool) string {
	if panicked {
		return "failed:internal"
	}
	status := w.Status()
	parsed := parseOpsErrorResponse(w.capturedBytes())
	if terminal, ok := w.capturedTerminalError(); ok {
		parsed = terminal
	}
	if parsed.StreamFailure {
		status = inferStreamFailureStatus(c, parsed)
	}
	for _, streamErr := range service.GetOpsStreamErrors(c) {
		status = streamErr.IntendedStatus
		if status < 400 {
			status = 502
		}
		parsed.ErrorType, parsed.Message = streamErr.ErrType, streamErr.Message
	}
	if service.GetOpsCyberPolicy(c) != nil {
		return "failed:content_policy"
	}
	if status == 499 {
		return "cancelled"
	}
	// A known terminal failure takes precedence over a subsequent disconnect.
	if status >= 400 {
		category := service.ClassifyChannelMonitorV2Error(service.ChannelMonitorV2ErrorInput{StatusCode: status, ErrorType: parsed.ErrorType, Message: parsed.Message})
		if category == "client_cancelled" {
			return "cancelled"
		}
		return "failed:" + category
	}
	if errors.Is(c.Request.Context().Err(), context.Canceled) {
		return "cancelled"
	}
	if errors.Is(c.Request.Context().Err(), context.DeadlineExceeded) {
		return "failed:timeout"
	}
	return "success"
}

func finishGroupRealtimeWSTurn(c *gin.Context, result *service.OpenAIForwardResult, err error) {
	outcome := "success"
	var retry *service.UpstreamFailoverError
	retryable := errors.As(err, &retry)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			outcome = "cancelled"
		} else {
			status := 502
			if retryable {
				status = retry.StatusCode
			}
			outcome = "failed:" + service.ClassifyChannelMonitorV2Error(service.ChannelMonitorV2ErrorInput{StatusCode: status, Message: err.Error()})
		}
	} else if result != nil && result.ClientDisconnect {
		outcome = "cancelled"
	} else if result == nil || !openAIForwardSucceededForScheduling(result) {
		outcome = "failed:transport_or_stream"
	}
	if service.GetOpsCyberPolicy(c) != nil {
		outcome = "failed:content_policy"
	}
	service.FinishGroupRealtimeTurn(c, outcome, retryable)
}
