// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"

	"github.com/jhuiting/terraform-provider-chargebee/internal/chargebee"
)

// identityField pairs a create-identity attribute name with whether the
// configuration supplied it.
type identityField struct {
	Name    string
	Present bool
}

// createIdentityMissing reports which identity attributes are missing when a
// configuration supplies some but not all of them. It returns nil when all are
// set (creating a new object) or none are set (adopting an existing object by
// id); a non-empty result means an invalid partial mix that must be rejected.
func createIdentityMissing(fields []identityField) []string {
	present := 0
	for _, f := range fields {
		if f.Present {
			present++
		}
	}
	if present == 0 || present == len(fields) {
		return nil
	}
	var missing []string
	for _, f := range fields {
		if !f.Present {
			missing = append(missing, f.Name)
		}
	}
	return missing
}

// adoptionConflict is a field whose value in an existing Chargebee object
// cannot be brought in line with the configuration during adoption.
type adoptionConflict struct {
	Field string
	Want  string
	Got   string
}

// adoptionField pairs a planned attribute value with the value produced after
// an adoption update, for post-update consistency verification.
type adoptionField struct {
	Name string
	Plan attr.Value
	Got  attr.Value
}

// adoptionConflictError builds the diagnostic shown when an existing Chargebee
// object cannot be adopted because some of its fields differ from the
// configuration. note explains why the listed fields could not be reconciled.
func adoptionConflictError(kind, id string, conflicts []adoptionConflict, note string) diag.Diagnostic {
	var b strings.Builder
	fmt.Fprintf(&b, "A %s with id %q already exists in Chargebee, but it cannot be adopted "+
		"because these fields differ from the configuration:\n", kind, id)
	for _, c := range conflicts {
		fmt.Fprintf(&b, "  - %s: configuration has %s, Chargebee has %s\n", c.Field, c.Want, c.Got)
	}
	b.WriteString("\n" + note)
	return diag.NewErrorDiagnostic(fmt.Sprintf("Cannot adopt existing %s", kind), b.String())
}

const (
	createOnlyConflictNote = "These fields cannot be changed after creation. Align the configuration with the " +
		"existing object, or delete the existing object in Chargebee before applying."
	unreconciledConflictNote = "Adoption could not make the existing object match the configuration for these " +
		"fields (for example, Chargebee does not allow clearing an optional field once set). Set them in the " +
		"configuration to match the existing object, or delete the existing object in Chargebee before applying."
)

// itemAdoptionConflicts reports create-only fields that differ between the
// planned item and the existing Chargebee item.
func itemAdoptionConflicts(plan, existing chargebee.Item) []adoptionConflict {
	var conflicts []adoptionConflict
	if plan.ItemFamilyID != existing.ItemFamilyID {
		conflicts = append(conflicts, adoptionConflict{"item_family_id", quote(plan.ItemFamilyID), quote(existing.ItemFamilyID)})
	}
	if plan.Type != existing.Type {
		conflicts = append(conflicts, adoptionConflict{"type", quote(plan.Type), quote(existing.Type)})
	}
	if plan.Metered != existing.Metered {
		conflicts = append(conflicts, adoptionConflict{"metered", fmt.Sprintf("%t", plan.Metered), fmt.Sprintf("%t", existing.Metered)})
	}
	if plan.UsageCalculation != existing.UsageCalculation {
		conflicts = append(conflicts, adoptionConflict{"usage_calculation", quote(plan.UsageCalculation), quote(existing.UsageCalculation)})
	}
	if boolValue(plan.IsGiftable) != boolValue(existing.IsGiftable) {
		conflicts = append(conflicts, adoptionConflict{"is_giftable", fmt.Sprintf("%t", boolValue(plan.IsGiftable)), fmt.Sprintf("%t", boolValue(existing.IsGiftable))})
	}
	return conflicts
}

// featureAdoptionConflicts reports create-only fields that differ between the
// planned feature and the existing Chargebee feature.
func featureAdoptionConflicts(plan, existing chargebee.Feature) []adoptionConflict {
	var conflicts []adoptionConflict
	if plan.Type != existing.Type {
		conflicts = append(conflicts, adoptionConflict{"type", quote(plan.Type), quote(existing.Type)})
	}
	return conflicts
}

// itemPriceAdoptionConflicts reports create-only fields that differ between the
// planned item price and the existing Chargebee item price.
func itemPriceAdoptionConflicts(plan, existing chargebee.ItemPrice) []adoptionConflict {
	var conflicts []adoptionConflict
	if plan.ItemID != existing.ItemID {
		conflicts = append(conflicts, adoptionConflict{"item_id", quote(plan.ItemID), quote(existing.ItemID)})
	}
	if plan.CurrencyCode != existing.CurrencyCode {
		conflicts = append(conflicts, adoptionConflict{"currency_code", quote(plan.CurrencyCode), quote(existing.CurrencyCode)})
	}
	return conflicts
}

// unreconciledFields reports fields whose known planned value differs from the
// value produced after an adoption update. Attributes still unknown in the
// plan (Chargebee-computed) are skipped, since any resulting value is valid.
func unreconciledFields(fields []adoptionField) []adoptionConflict {
	var conflicts []adoptionConflict
	for _, f := range fields {
		if f.Plan.IsUnknown() || f.Plan.Equal(f.Got) {
			continue
		}
		conflicts = append(conflicts, adoptionConflict{f.Name, f.Plan.String(), f.Got.String()})
	}
	return conflicts
}

func boolValue(p *bool) bool { return p != nil && *p }

func quote(s string) string { return fmt.Sprintf("%q", s) }
