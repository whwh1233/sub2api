package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
)

// GetDailyLeaderboard returns the daily token top list plus the current user's row when present.
func (r *usageLogRepository) GetDailyLeaderboard(ctx context.Context, startTime, endTime time.Time, currentUserID int64, limit int) (result *usagestats.DailyLeaderboardResponse, err error) {
	if limit <= 0 {
		limit = 3
	}

	query := `
		WITH daily_usage AS (
			SELECT
				ul.user_id,
				COALESCE(NULLIF(TRIM(u.username), ''), '') AS username,
				COALESCE(NULLIF(TRIM(u.email), ''), '') AS email,
				COUNT(*)::BIGINT AS requests,
				COALESCE(SUM(
					COALESCE(ul.input_tokens, 0) +
					COALESCE(ul.output_tokens, 0) +
					COALESCE(ul.cache_creation_tokens, 0) +
					COALESCE(ul.cache_read_tokens, 0)
				), 0)::BIGINT AS total_tokens
			FROM usage_logs ul
			LEFT JOIN users u ON ul.user_id = u.id
			WHERE ul.created_at >= $1 AND ul.created_at < $2
				AND ul.user_id IS NOT NULL
			GROUP BY ul.user_id, u.username, u.email
		),
		ranked AS (
			SELECT
				ROW_NUMBER() OVER (ORDER BY total_tokens DESC, requests DESC, user_id ASC)::BIGINT AS rank,
				user_id,
				username,
				email,
				requests,
				total_tokens
			FROM daily_usage
		)
		SELECT rank, user_id, username, email, requests, total_tokens
		FROM ranked
		WHERE rank <= $3 OR user_id = $4
		ORDER BY rank ASC
	`

	rows, err := r.sql.QueryContext(ctx, query, startTime, endTime, limit, currentUserID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = closeErr
			result = nil
		}
	}()

	result = &usagestats.DailyLeaderboardResponse{
		Top: make([]usagestats.DailyLeaderboardItem, 0, limit),
		Me: usagestats.DailyLeaderboardMe{
			DailyLeaderboardItem: usagestats.DailyLeaderboardItem{
				UserID:        currentUserID,
				DisplayName:   dailyLeaderboardDisplayName(currentUserID, "", ""),
				IsCurrentUser: true,
			},
		},
	}

	for rows.Next() {
		var (
			rank        int64
			userID      int64
			username    string
			email       string
			requests    int64
			totalTokens int64
		)
		if err = rows.Scan(&rank, &userID, &username, &email, &requests, &totalTokens); err != nil {
			return nil, err
		}
		rankCopy := rank
		item := usagestats.DailyLeaderboardItem{
			Rank:          &rankCopy,
			UserID:        userID,
			DisplayName:   dailyLeaderboardDisplayName(userID, username, email),
			TotalTokens:   totalTokens,
			Requests:      requests,
			IsCurrentUser: userID == currentUserID,
		}
		if rank <= int64(limit) {
			result.Top = append(result.Top, item)
		}
		if userID == currentUserID {
			result.Me.DailyLeaderboardItem = item
			result.Me.IsCurrentUser = true
		}
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	result.Me.TokensToTopThree = dailyLeaderboardGapToTopThree(result.Top, result.Me)
	return result, nil
}

func dailyLeaderboardGapToTopThree(top []usagestats.DailyLeaderboardItem, me usagestats.DailyLeaderboardMe) int64 {
	const targetRank = 3
	if me.Rank != nil && *me.Rank <= targetRank {
		return 0
	}
	if len(top) < targetRank {
		if me.TotalTokens > 0 {
			return 0
		}
		return 1
	}
	target := top[targetRank-1].TotalTokens + 1
	if target <= me.TotalTokens {
		return 0
	}
	return target - me.TotalTokens
}

func dailyLeaderboardDisplayName(userID int64, username, email string) string {
	username = strings.TrimSpace(username)
	if username != "" {
		return username
	}
	email = strings.TrimSpace(email)
	if email != "" {
		return maskLeaderboardEmail(email)
	}
	return fmt.Sprintf("User #%d", userID)
}

func maskLeaderboardEmail(email string) string {
	at := strings.Index(email, "@")
	if at <= 0 {
		return maskLeaderboardEmailPart(email)
	}
	local := maskLeaderboardEmailPart(email[:at])
	domain := email[at+1:]
	if domain == "" {
		return local + "@***"
	}
	labels := strings.Split(domain, ".")
	for i, label := range labels {
		if label == "" {
			labels[i] = "***"
			continue
		}
		normalized := strings.ToLower(label)
		if normalized == "com" || normalized == "cn" {
			labels[i] = label
			continue
		}
		labels[i] = maskLeaderboardEmailPart(label)
	}
	return local + "@" + strings.Join(labels, ".")
}

func maskLeaderboardEmailPart(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "***"
	}
	runes := []rune(value)
	return string(runes[0]) + "***"
}
