package handler

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGroupRealtimeOutcome(t *testing.T) {
	for _, tc := range []struct {
		name              string
		status            int
		body              string
		cancelled, marked bool
		want              string
	}{
		{"success", 200, `{"ok":true}`, false, false, "success"},
		{"quota", 403, `{"error":{"message":"insufficient balance"}}`, false, false, "failed:quota_or_balance"},
		{"stream failure", 200, "event: response.failed\ndata: {\"type\":\"response.failed\",\"response\":{\"error\":{\"message\":\"upstream failed\"}}}\n\n", false, false, "failed:internal"},
		{"cancelled", 200, "", true, false, "cancelled"},
		{"marked stream", 200, "", false, true, "failed:timeout"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("POST", "/v1/responses", nil)
			if tc.cancelled {
				ctx, cancel := context.WithCancel(c.Request.Context())
				cancel()
				c.Request = c.Request.WithContext(ctx)
			}
			w := acquireOpsCaptureWriter(c.Writer)
			defer releaseOpsCaptureWriter(w)
			w.setContext(c)
			c.Writer = w
			if tc.name == "stream failure" {
				w.Header().Set("Content-Type", "text/event-stream")
			}
			w.WriteHeader(tc.status)
			_, err := w.WriteString(tc.body)
			require.NoError(t, err)
			w.finalizeCapture()
			if tc.marked {
				service.MarkOpsStreamFailure(c, "timeout", "timeout", "deadline exceeded", 504)
			}
			require.Equal(t, tc.want, groupRealtimeOutcome(c, w, false))
		})
	}
}

func TestGroupRealtimeRequestScope(t *testing.T) {
	for path, want := range map[string]bool{"/v1/responses": true, "/v1/messages": true, "/v1beta/models/gemini:streamGenerateContent": true, "/v1/messages/count_tokens": false, "/v1/responses/input_tokens": false, "/v1/usage": false} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("POST", path, nil)
		require.Equal(t, want, isGroupRealtimeRequest(c), path)
	}
}
