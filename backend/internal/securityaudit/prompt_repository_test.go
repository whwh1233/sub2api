package securityaudit

import (
	"strings"
	"testing"
)

func TestPromptEventUpstreamStatusFilter(t *testing.T) {
	filter := EventFilter{RequestID: "request-123", UpstreamStatus: 403}
	where, args := buildEventWhere(filter, 2)
	if len(args) != 2 || args[1] != 403 || !strings.Contains(where, "o.upstream_status_code=$3") {
		t.Fatalf("unexpected filter: %s %#v", where, args)
	}
	for _, clause := range []string{"EXISTS", "o.request_id=e.request_id", "o.user_id IS NOT DISTINCT FROM e.user_id", "o.api_key_id IS NOT DISTINCT FROM e.api_key_id", "e.request_id<>''"} {
		if !strings.Contains(where, clause) {
			t.Fatalf("missing correlation guard %q", clause)
		}
	}
	all := filter
	all.UpstreamStatus = 0
	where, _ = buildEventWhere(all, 1)
	if strings.Contains(where, "ops_error_logs") {
		t.Fatal("default search must not query upstream logs")
	}
	if FilterHash(filter, 1) == FilterHash(all, 1) {
		t.Fatal("delete confirmation must include upstream status")
	}
}

func TestPromptEventModelFilterCombinesWithGroup(t *testing.T) {
	group := int64(7)
	filter := EventFilter{GroupID: &group, Model: " opus_5% "}
	where, args := buildEventWhere(filter, 1)
	if !strings.Contains(where, "e.model ILIKE $1") || !strings.Contains(where, "e.group_id=$2") {
		t.Fatalf("missing combined filters: %s", where)
	}
	if len(args) != 2 || args[0] != `%opus\_5\%%` || args[1] != group {
		t.Fatalf("unexpected escaped model/group arguments: %#v", args)
	}
	withoutModel := filter
	withoutModel.Model = ""
	if FilterHash(filter, 1) == FilterHash(withoutModel, 1) {
		t.Fatal("delete confirmation must include model")
	}
}

func TestShouldStorePromptAuditEvent(t *testing.T) {
	tests := []struct {
		name            string
		storePassEvents bool
		decision        EventDecision
		want            bool
	}{
		{name: "pass disabled", storePassEvents: false, decision: EventPass, want: false},
		{name: "flag disabled", storePassEvents: false, decision: EventFlag, want: true},
		{name: "critical disabled", storePassEvents: false, decision: EventCritical, want: true},
		{name: "pass enabled", storePassEvents: true, decision: EventPass, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldStorePromptAuditEvent(tt.decision, tt.storePassEvents); got != tt.want {
				t.Fatalf("shouldStorePromptAuditEvent(%q, %t) = %t, want %t", tt.decision, tt.storePassEvents, got, tt.want)
			}
		})
	}
}
