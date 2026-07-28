// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"github.com/jhuiting/terraform-provider-chargebee/internal/chargebee"
)

// itemPriceConfigErrors validates the cross-attribute rules of a
// chargebee_item_price configuration. Unknown values (e.g. from variables not
// yet resolved) must be passed as "not set" so validation stays permissive
// until apply time.
type itemPriceConfig struct {
	PricingModel  string
	HasPrice      bool
	TierCount     int
	HasPeriod     bool
	HasPeriodUnit bool
}

type itemPriceConfigError struct {
	Attribute string
	Summary   string
	Detail    string
}

func itemPriceConfigErrors(c itemPriceConfig) []itemPriceConfigError {
	var errs []itemPriceConfigError

	switch c.PricingModel {
	case chargebee.PricingModelFlatFee, chargebee.PricingModelPerUnit:
		if !c.HasPrice {
			errs = append(errs, itemPriceConfigError{
				Attribute: "price",
				Summary:   "price is required",
				Detail:    "Pricing models flat_fee and per_unit require the price attribute.",
			})
		}
		if c.TierCount > 0 {
			errs = append(errs, itemPriceConfigError{
				Attribute: "tiers",
				Summary:   "tiers not allowed",
				Detail:    "Tiers are only valid for the tiered, volume, and stairstep pricing models.",
			})
		}
	case chargebee.PricingModelTiered, chargebee.PricingModelVolume, chargebee.PricingModelStairstep:
		if c.TierCount == 0 {
			errs = append(errs, itemPriceConfigError{
				Attribute: "tiers",
				Summary:   "tiers are required",
				Detail:    "Pricing models tiered, volume, and stairstep require at least one tier.",
			})
		}
		if c.HasPrice {
			errs = append(errs, itemPriceConfigError{
				Attribute: "price",
				Summary:   "price not allowed",
				Detail:    "The price attribute is only valid for the flat_fee and per_unit pricing models; use tiers instead.",
			})
		}
	}

	if c.HasPeriod != c.HasPeriodUnit {
		errs = append(errs, itemPriceConfigError{
			Attribute: "period",
			Summary:   "period and period_unit must be set together",
			Detail:    "Set both period and period_unit for recurring prices, or neither for one-time charges.",
		})
	}

	return errs
}
