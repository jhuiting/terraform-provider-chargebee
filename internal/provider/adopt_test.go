// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"

	"github.com/jhuiting/terraform-provider-chargebee/internal/chargebee"
)

func TestItemAdoptionConflicts(t *testing.T) {
	base := chargebee.Item{
		ID:               "eye",
		ItemFamilyID:     "fam",
		Type:             "plan",
		Metered:          false,
		UsageCalculation: "",
		IsGiftable:       boolPtr(false),
	}

	tests := []struct {
		name       string
		plan       chargebee.Item
		existing   chargebee.Item
		wantFields []string
	}{
		{"identical", base, base, nil},
		{
			"family differs",
			base,
			func() chargebee.Item { e := base; e.ItemFamilyID = "other"; return e }(),
			[]string{"item_family_id"},
		},
		{
			"type differs",
			base,
			func() chargebee.Item { e := base; e.Type = "addon"; return e }(),
			[]string{"type"},
		},
		{
			"metered and giftable differ",
			func() chargebee.Item { p := base; p.Metered = true; p.IsGiftable = boolPtr(true); return p }(),
			base,
			[]string{"metered", "is_giftable"},
		},
		{
			"nil giftable equals false giftable",
			func() chargebee.Item { p := base; p.IsGiftable = nil; return p }(),
			base,
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conflicts := itemAdoptionConflicts(tt.plan, tt.existing)
			assert.Equal(t, tt.wantFields, conflictFields(conflicts))
		})
	}
}

func TestFeatureAdoptionConflicts(t *testing.T) {
	base := chargebee.Feature{ID: "sso", Name: "Single sign-on", Type: "switch"}

	tests := []struct {
		name       string
		plan       chargebee.Feature
		existing   chargebee.Feature
		wantFields []string
	}{
		{"identical", base, base, nil},
		{
			"type differs",
			base,
			func() chargebee.Feature { e := base; e.Type = "quantity"; return e }(),
			[]string{"type"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conflicts := featureAdoptionConflicts(tt.plan, tt.existing)
			assert.Equal(t, tt.wantFields, conflictFields(conflicts))
		})
	}
}

func TestItemPriceAdoptionConflicts(t *testing.T) {
	base := chargebee.ItemPrice{ID: "eye-EUR", ItemID: "eye", CurrencyCode: "EUR"}

	tests := []struct {
		name       string
		plan       chargebee.ItemPrice
		existing   chargebee.ItemPrice
		wantFields []string
	}{
		{"identical", base, base, nil},
		{
			"item and currency differ",
			base,
			func() chargebee.ItemPrice { e := base; e.ItemID = "other"; e.CurrencyCode = "USD"; return e }(),
			[]string{"item_id", "currency_code"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conflicts := itemPriceAdoptionConflicts(tt.plan, tt.existing)
			assert.Equal(t, tt.wantFields, conflictFields(conflicts))
		})
	}
}

func TestAdoptionConflictError(t *testing.T) {
	err := adoptionConflictError("item", "eye", []adoptionConflict{
		{"type", `"plan"`, `"addon"`},
	}, createOnlyConflictNote)
	assert.Contains(t, err.Summary(), "Cannot adopt existing item")
	assert.Contains(t, err.Detail(), `id "eye"`)
	assert.Contains(t, err.Detail(), "type")
	assert.Contains(t, err.Detail(), "addon")
	assert.Contains(t, err.Detail(), "cannot be changed after creation")
}

func TestUnreconciledFields(t *testing.T) {
	tests := []struct {
		name       string
		fields     []adoptionField
		wantFields []string
	}{
		{
			"all match",
			[]adoptionField{
				{"name", types.StringValue("a"), types.StringValue("a")},
				{"description", types.StringNull(), types.StringNull()},
			},
			nil,
		},
		{
			"description could not be cleared",
			[]adoptionField{
				{"name", types.StringValue("a"), types.StringValue("a")},
				{"description", types.StringNull(), types.StringValue("leftover")},
			},
			[]string{"description"},
		},
		{
			"unknown plan value is skipped",
			[]adoptionField{
				{"external_name", types.StringUnknown(), types.StringValue("server default")},
			},
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantFields, conflictFields(unreconciledFields(tt.fields)))
		})
	}
}

func TestCreateIdentityMissing(t *testing.T) {
	f := func(name string, present bool) identityField { return identityField{name, present} }

	tests := []struct {
		name        string
		fields      []identityField
		wantMissing []string
	}{
		{"all set (create)", []identityField{f("name", true), f("type", true), f("item_family_id", true)}, nil},
		{"none set (adopt by id)", []identityField{f("name", false), f("type", false), f("item_family_id", false)}, nil},
		{"name only", []identityField{f("name", true), f("type", false), f("item_family_id", false)}, []string{"type", "item_family_id"}},
		{"missing one of four", []identityField{f("item_id", true), f("name", true), f("currency_code", true), f("pricing_model", false)}, []string{"pricing_model"}},
		{"missing two of four", []identityField{f("item_id", true), f("name", false), f("currency_code", true), f("pricing_model", false)}, []string{"name", "pricing_model"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantMissing, createIdentityMissing(tt.fields))
		})
	}
}

func conflictFields(conflicts []adoptionConflict) []string {
	if len(conflicts) == 0 {
		return nil
	}
	fields := make([]string, len(conflicts))
	for i, c := range conflicts {
		fields[i] = c.Field
	}
	return fields
}

func boolPtr(b bool) *bool { return &b }
