package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/stretchr/testify/require"
)

type usageLeaderboardCacheRepoStub struct {
	UsageLogRepository
	calls int
}

func (s *usageLeaderboardCacheRepoStub) GetDailyLeaderboard(
	ctx context.Context,
	startTime, endTime time.Time,
	currentUserID int64,
	limit int,
) (*usagestats.DailyLeaderboardResponse, error) {
	s.calls++
	rank := int64(s.calls)
	return &usagestats.DailyLeaderboardResponse{
		Top: []usagestats.DailyLeaderboardItem{
			{
				Rank:        &rank,
				UserID:      currentUserID,
				DisplayName: "cached-user",
				TotalTokens: int64(100 * s.calls),
				Requests:    int64(s.calls),
			},
		},
		Me: usagestats.DailyLeaderboardMe{
			DailyLeaderboardItem: usagestats.DailyLeaderboardItem{
				Rank:        &rank,
				UserID:      currentUserID,
				DisplayName: "cached-user",
				TotalTokens: int64(100 * s.calls),
				Requests:    int64(s.calls),
			},
		},
	}, nil
}

func TestUsageServiceDailyLeaderboardCachesPerUserForOneMinute(t *testing.T) {
	require.NoError(t, timezone.Init("Asia/Shanghai"))
	t.Cleanup(func() { _ = timezone.Init("UTC") })

	repo := &usageLeaderboardCacheRepoStub{}
	svc := NewUsageService(repo, nil, nil, nil)

	now := time.Date(2026, 5, 30, 10, 0, 0, 0, time.Local)
	svc.now = func() time.Time { return now }

	first, err := svc.GetDailyLeaderboard(context.Background(), 42)
	require.NoError(t, err)
	second, err := svc.GetDailyLeaderboard(context.Background(), 42)
	require.NoError(t, err)

	require.Equal(t, 1, repo.calls)
	require.Equal(t, first.Top[0].TotalTokens, second.Top[0].TotalTokens)

	now = now.Add(time.Minute + time.Nanosecond)
	third, err := svc.GetDailyLeaderboard(context.Background(), 42)
	require.NoError(t, err)

	require.Equal(t, 2, repo.calls)
	require.NotEqual(t, second.Top[0].TotalTokens, third.Top[0].TotalTokens)
}

func TestUsageServiceDailyLeaderboardCacheIsScopedByUser(t *testing.T) {
	require.NoError(t, timezone.Init("Asia/Shanghai"))
	t.Cleanup(func() { _ = timezone.Init("UTC") })

	repo := &usageLeaderboardCacheRepoStub{}
	svc := NewUsageService(repo, nil, nil, nil)
	now := time.Date(2026, 5, 30, 10, 0, 0, 0, time.Local)
	svc.now = func() time.Time { return now }

	_, err := svc.GetDailyLeaderboard(context.Background(), 42)
	require.NoError(t, err)
	_, err = svc.GetDailyLeaderboard(context.Background(), 43)
	require.NoError(t, err)

	require.Equal(t, 2, repo.calls)
}

func TestUsageServiceDailyLeaderboardCachePrunesExpiredEntries(t *testing.T) {
	require.NoError(t, timezone.Init("Asia/Shanghai"))
	t.Cleanup(func() { _ = timezone.Init("UTC") })

	repo := &usageLeaderboardCacheRepoStub{}
	svc := NewUsageService(repo, nil, nil, nil)
	now := time.Date(2026, 5, 30, 10, 0, 0, 0, time.Local)
	svc.now = func() time.Time { return now }

	_, err := svc.GetDailyLeaderboard(context.Background(), 42)
	require.NoError(t, err)
	require.Len(t, svc.dailyLeaderboardCacheByID, 1)

	now = now.Add(time.Minute + time.Nanosecond)
	_, err = svc.GetDailyLeaderboard(context.Background(), 43)
	require.NoError(t, err)

	require.Len(t, svc.dailyLeaderboardCacheByID, 1)
	for key := range svc.dailyLeaderboardCacheByID {
		require.Equal(t, int64(43), key.userID)
	}
}
