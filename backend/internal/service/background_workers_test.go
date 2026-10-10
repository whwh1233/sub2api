package service

import (
	"context"
	"testing"
	"time"
)

type validationExpiryRepository struct {
	AccountRepository
	calls chan struct{}
}

func TestClaudeCodeVersionSyncProviderDisabledForValidation(t *testing.T) {
	t.Setenv("SERVER_DISABLE_BACKGROUND_WORKERS", "true")
	repo := newClaudeCodeVersionSyncSettingRepoStub(nil)
	client := &claudeCodeVersionSyncGitHubStub{
		latest: &GitHubRelease{TagName: "v2.1.999"},
	}
	svc := ProvideClaudeCodeVersionSyncService(repo, &SettingService{}, client)
	// Stop joins any accidentally started initial sync before checking effects.
	svc.Stop()
	if client.latestCalls != 0 || client.calls != 0 || len(repo.syncedWrites()) != 0 {
		t.Fatal("local validation started Claude version synchronization")
	}
}

func (r *validationExpiryRepository) AutoPauseExpiredAccounts(context.Context, time.Time) (int64, error) {
	r.calls <- struct{}{}
	return 0, nil
}

// Exercise the real provider: a clone must not mutate accounts on startup,
// while an ordinary production startup must retain the immediate expiry pass.
func TestAccountExpiryProviderBackgroundWorkerPolicy(t *testing.T) {
	for _, disabled := range []bool{true, false} {
		name, value := "production", "false"
		if disabled {
			name, value = "validation", "true"
		}
		t.Run(name, func(t *testing.T) {
			t.Setenv("SERVER_DISABLE_BACKGROUND_WORKERS", value)
			repo := &validationExpiryRepository{calls: make(chan struct{}, 1)}
			svc := ProvideAccountExpiryService(repo)
			defer svc.Stop()
			if disabled {
				// Stop joins a started worker, so an accidental startup cannot
				// escape detection through goroutine scheduling.
				svc.Stop()
				select {
				case <-repo.calls:
					t.Fatal("validation startup modified accounts")
				default:
				}
				return
			}
			select {
			case <-repo.calls:
			case <-time.After(3 * time.Second):
				t.Fatal("production startup did not run account expiry")
			}
		})
	}
}
