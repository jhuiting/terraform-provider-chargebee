// SPDX-License-Identifier: MPL-2.0

package chargebee

// Object statuses returned by the Chargebee API.
const (
	StatusActive   = "active"
	StatusArchived = "archived"
	StatusDeleted  = "deleted"
)

// Pricing models supported by Chargebee item prices.
const (
	PricingModelFlatFee   = "flat_fee"
	PricingModelPerUnit   = "per_unit"
	PricingModelTiered    = "tiered"
	PricingModelVolume    = "volume"
	PricingModelStairstep = "stairstep"
)

// ItemFamily is a Chargebee PC 2.0 product family.
type ItemFamily struct {
	ID          string
	Name        string
	Description string
	Status      string
}

// Item is a Chargebee PC 2.0 item (plan, addon, or charge).
type Item struct {
	ID                 string
	ItemFamilyID       string
	Type               string
	Name               string
	ExternalName       string
	Description        string
	Unit               string
	RedirectURL        string
	Metered            bool
	UsageCalculation   string
	EnabledForCheckout *bool
	EnabledInPortal    *bool
	IsShippable        *bool
	IsGiftable         *bool
	Metadata           map[string]string
	Status             string
}

// AttachedItem links a child item (addon or charge) to a parent plan item.
type AttachedItem struct {
	ID            string
	ParentItemID  string
	ItemID        string
	Type          string
	ChargeOnEvent string
	ChargeOnce    *bool
	Quantity      *int32
	BillingCycles *int32
	Status        string
}

// Feature types supported by Chargebee.
const (
	FeatureTypeSwitch   = "switch"
	FeatureTypeCustom   = "custom"
	FeatureTypeQuantity = "quantity"
	FeatureTypeRange    = "range"
)

// Feature is a Chargebee entitlement feature. Newly created features start in
// draft and must be activated before an item can be granted them.
type Feature struct {
	ID          string
	Name        string
	Description string
	Type        string
	Unit        string
	Status      string
}

// ItemEntitlement grants a feature to an item (plan, addon, or charge). For a
// switch feature the value is "true"; quantity and range features carry a
// numeric value. ItemType is the entity type Chargebee's Entitlements API keys
// the grant by (plan, addon, or charge) and must match the item's type.
type ItemEntitlement struct {
	ItemID    string
	ItemType  string
	FeatureID string
	Value     string
}

// Tier is one quantity bracket of a tiered, volume, or stairstep price.
type Tier struct {
	StartingUnit int32
	EndingUnit   *int32
	Price        int64
}

// ItemPrice is a Chargebee PC 2.0 item price.
type ItemPrice struct {
	ID                        string
	ItemID                    string
	Name                      string
	ExternalName              string
	Description               string
	InvoiceNotes              string
	CurrencyCode              string
	PricingModel              string
	Price                     *int64
	Tiers                     []Tier
	Period                    *int32
	PeriodUnit                string
	TrialPeriod               *int32
	TrialPeriodUnit           string
	BillingCycles             *int32
	FreeQuantity              *int32
	IsTaxable                 *bool
	ShowDescriptionInInvoices *bool
	ShowDescriptionInQuotes   *bool
	Status                    string
}
