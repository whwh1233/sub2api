//go:build unit

package service

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestUpgradeFableExplicitMaxTariffPreservesCharge(t *testing.T) {
	raw, err := os.ReadFile("../../../deploy/release-pricing-v0.2.15.json")
	require.NoError(t, err)
	var plan struct {
		Card ChannelModelPricing `json:"after_upgrade_card"`
	}
	require.NoError(t, json.Unmarshal(raw, &plan))
	bs := newTestBillingService()
	resolver := NewModelPricingResolver(nil, bs)
	group := &Group{ID: 7, RateMultiplier: 1.5, LongContextPricingEnabled: true, ModelPricing: []ChannelModelPricing{plan.Card}}
	auth := &APIKeyService{}
	apiKey := &APIKey{ID: 82, UserID: 40, GroupID: &group.ID, Status: StatusActive,
		User: &User{ID: 40, Status: StatusActive}, Group: group}
	payload, err := json.Marshal(&APIKeyAuthCacheEntry{Snapshot: auth.snapshotFromAPIKey(context.Background(), apiKey)})
	require.NoError(t, err)
	var cached APIKeyAuthCacheEntry
	require.NoError(t, json.Unmarshal(payload, &cached))
	materialized, used, err := auth.applyAuthCacheEntry("pricing-tariff-roundtrip", &cached)
	require.NoError(t, err)
	require.True(t, used)
	group = materialized.Group
	for _, tc := range []struct {
		model, effort string
		want          float64
	}{
		{"claude-fable-5-1", "max", 0.43480575},
		{"claude-fable-5-1-20261001", "max", 0.43480575},
		{"claude-fable-5-1", "high", 0.14493525},
	} {
		t.Run(tc.model+"/"+tc.effort, func(t *testing.T) {
			cost, err := bs.CalculateCostUnified(CostInput{
				Ctx: context.Background(), Model: tc.model, Group: group, GroupID: &group.ID,
				ReasoningEffort: tc.effort, RateMultiplier: group.RateMultiplier, Resolver: resolver,
				Tokens: UsageTokens{InputTokens: 506, OutputTokens: 12, CacheCreationTokens: 1011, CacheCreation5mTokens: 1011, CacheReadTokens: 313304},
			})
			require.NoError(t, err)
			require.InDelta(t, tc.want, cost.ActualCost, 1e-10)
		})
	}
	base, err := bs.GetModelPricing("claude-fable-5-1")
	require.NoError(t, err)
	require.Empty(t, base.ReasoningEffortMultipliers, "group override must not mutate shared catalog pricing")
}

func TestUpgradeHaikuLongContextTariffIsSeparateFromMax(t *testing.T) {
	bs := NewBillingService(&config.Config{}, newStubPricingServiceFromMap(map[string]*LiteLLMModelPricing{
		"claude-haiku-5-5": {
			InputCostPerToken: 1e-7, OutputCostPerToken: 5e-7,
			CacheCreationInputTokenCost: 1.25e-7, CacheCreationInputTokenCostAbove1hr: 2e-7,
			CacheReadInputTokenCost: 1e-8, LongContextInputTokenThreshold: 100000,
			LongContextInputCostMultiplier: 5, LongContextOutputCostMultiplier: 5,
		},
	}))
	for _, enabled := range []bool{true, false} {
		group := &Group{ID: 7, LongContextPricingEnabled: enabled, ModelPricing: []ChannelModelPricing{{
			Models: []string{"claude-fable-5-1"}, BillingMode: BillingModeToken,
			ReasoningEffortMultipliers: map[string]float64{"max": 3},
		}}}
		cost, err := bs.CalculateCostUnified(CostInput{
			Ctx: context.Background(), Model: "claude-haiku-5-5", Group: group, GroupID: &group.ID,
			RateMultiplier: 1.25, Resolver: NewModelPricingResolver(nil, bs),
			Tokens: UsageTokens{InputTokens: 4, OutputTokens: 63, CacheCreationTokens: 132530, CacheCreation5mTokens: 132530},
		})
		require.NoError(t, err)
		want := 0.1037384375
		if !enabled {
			want /= 5
		}
		require.InDelta(t, want, cost.ActualCost, 1e-10)
		require.Equal(t, enabled, cost.LongContextBillingApplied)
	}
}
