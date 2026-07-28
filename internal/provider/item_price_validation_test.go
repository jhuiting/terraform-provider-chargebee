// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestItemPriceConfigErrors(t *testing.T) {
	tests := []struct {
		name           string
		config         itemPriceConfig
		wantAttributes []string
	}{
		{
			name:   "valid flat_fee",
			config: itemPriceConfig{PricingModel: "flat_fee", HasPrice: true, HasPeriod: true, HasPeriodUnit: true},
		},
		{
			name:   "valid per_unit one-time",
			config: itemPriceConfig{PricingModel: "per_unit", HasPrice: true},
		},
		{
			name:   "valid tiered",
			config: itemPriceConfig{PricingModel: "tiered", TierCount: 2, HasPeriod: true, HasPeriodUnit: true},
		},
		{
			name:   "unknown pricing model is permissive",
			config: itemPriceConfig{PricingModel: ""},
		},
		{
			name:           "flat_fee without price",
			config:         itemPriceConfig{PricingModel: "flat_fee"},
			wantAttributes: []string{"price"},
		},
		{
			name:           "per_unit with tiers",
			config:         itemPriceConfig{PricingModel: "per_unit", HasPrice: true, TierCount: 1},
			wantAttributes: []string{"tiers"},
		},
		{
			name:           "volume without tiers",
			config:         itemPriceConfig{PricingModel: "volume"},
			wantAttributes: []string{"tiers"},
		},
		{
			name:           "stairstep with price and no tiers",
			config:         itemPriceConfig{PricingModel: "stairstep", HasPrice: true},
			wantAttributes: []string{"tiers", "price"},
		},
		{
			name:           "period without period_unit",
			config:         itemPriceConfig{PricingModel: "flat_fee", HasPrice: true, HasPeriod: true},
			wantAttributes: []string{"period"},
		},
		{
			name:           "period_unit without period",
			config:         itemPriceConfig{PricingModel: "flat_fee", HasPrice: true, HasPeriodUnit: true},
			wantAttributes: []string{"period"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := itemPriceConfigErrors(tt.config)
			var got []string
			for _, e := range errs {
				got = append(got, e.Attribute)
				assert.NotEmpty(t, e.Summary)
				assert.NotEmpty(t, e.Detail)
			}
			assert.ElementsMatch(t, tt.wantAttributes, got)
		})
	}
}
