package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAPIKeyAuthSnapshotGroupPricingRoundtrip(t *testing.T) {
	groupID := int64(50)
	inputPrice := 1e-6
	outputPrice := 2e-6
	apiKey := &APIKey{
		ID: 82, UserID: 40, GroupID: &groupID, Key: "sk-pricing-roundtrip", Status: StatusActive,
		User: &User{ID: 40, Status: StatusActive},
		Group: &Group{
			ID: groupID, Name: "pricing-roundtrip", Platform: PlatformAnthropic, Status: StatusActive,
			LongContextPricingEnabled: true,
			ModelPricing: []ChannelModelPricing{{
				Models: []string{"claude-sonnet-*"}, BillingMode: BillingModeToken,
				InputPrice: &inputPrice, OutputPrice: &outputPrice,
				ReasoningEffortMultipliers: map[string]float64{"max": 3},
			}},
		},
	}
	svc := &APIKeyService{}

	payload, err := json.Marshal(&APIKeyAuthCacheEntry{Snapshot: svc.snapshotFromAPIKey(context.Background(), apiKey)})
	require.NoError(t, err)
	var cached APIKeyAuthCacheEntry
	require.NoError(t, json.Unmarshal(payload, &cached))

	materialized, used, err := svc.applyAuthCacheEntry(apiKey.Key, &cached)
	require.NoError(t, err)
	require.True(t, used)
	require.NotNil(t, materialized.Group)
	require.True(t, materialized.Group.LongContextPricingEnabled)
	require.Equal(t, apiKey.Group.ModelPricing, materialized.Group.ModelPricing)

	billing := &BillingService{fallbackPrices: map[string]*ModelPricing{
		"claude-sonnet-4": {InputPricePerToken: 3e-6, OutputPricePerToken: 15e-6},
	}}
	resolver := NewModelPricingResolver(nil, billing)
	resolved := resolver.Resolve(context.Background(), PricingInput{Model: "claude-sonnet-4", Group: materialized.Group})
	require.Equal(t, PricingSourceGroup, resolved.Source)
	require.True(t, resolved.longContextPricingEnabled)
	require.InDelta(t, inputPrice, resolved.BasePricing.InputPricePerToken, 1e-12)
	require.InDelta(t, outputPrice, resolved.BasePricing.OutputPricePerToken, 1e-12)
}

func TestAPIKeyAuthSnapshotRejectsLegacyMaxPricing(t *testing.T) {
	// A v0.2.4 Redis entry carries the old field. Decoding it into the new
	// pricing type silently drops the multiplier; it must force a DB reload.
	var cached APIKeyAuthCacheEntry
	require.NoError(t, json.Unmarshal([]byte(`{
	  "snapshot": {
	    "version": 24,
	    "api_key_id": 82,
	    "user_id": 40,
	    "group_id": 7,
	    "status": "active",
	    "user": {"id": 40, "status": "active"},
	    "group": {"id": 7, "platform": "anthropic", "status": "active",
	      "model_pricing": [{"models": ["claude-fable-5-1"], "billing_mode": "token", "max_reasoning_effort_multiplier": 3}]}
	  }
	}`), &cached))
	require.NotNil(t, cached.Snapshot)
	require.Empty(t, cached.Snapshot.Group.ModelPricing[0].ReasoningEffortMultipliers)
	svc := &APIKeyService{}
	key, used, err := svc.applyAuthCacheEntry("legacy-pricing-test", &cached)
	require.NoError(t, err)
	require.False(t, used, "legacy pricing must be reloaded from the migrated database")
	require.Nil(t, key)
}
