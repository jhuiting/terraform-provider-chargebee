// SPDX-License-Identifier: MPL-2.0

package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccItemResource(t *testing.T) {
	familyID := testAccRandomID("tf-acc-family")
	itemID := testAccRandomID("tf-acc-item")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccItemConfig(familyID, itemID, "addon", "Extra Seats "+itemID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("chargebee_item.test", "id", itemID),
					resource.TestCheckResourceAttr("chargebee_item.test", "type", "addon"),
					resource.TestCheckResourceAttr("chargebee_item.test", "metered", "false"),
					resource.TestCheckResourceAttr("chargebee_item.test", "status", "active"),
				),
			},
			{
				Config: testAccItemConfig(familyID, itemID, "addon", "Renamed "+itemID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("chargebee_item.test", "name", "Renamed "+itemID),
				),
			},
			{
				// Changing the type must force replacement.
				Config: testAccItemConfig(familyID, itemID, "charge", "Renamed "+itemID),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("chargebee_item.test", plancheck.ResourceActionReplace),
					},
				},
			},
			{
				ResourceName:      "chargebee_item.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccItemResourceInvalidUsageCalculation(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "chargebee_item" "test" {
  id                = "never-created"
  item_family_id    = "whatever"
  type              = "plan"
  name              = "Invalid"
  usage_calculation = "sum_of_usages"
}
`,
				ExpectError: regexp.MustCompile(`usage_calculation requires metered = true`),
			},
		},
	})
}

func TestAccItemResourceAdoptByIDNoOp(t *testing.T) {
	familyID := testAccRandomID("tf-acc-family")
	itemID := testAccRandomID("tf-acc-item")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccItemConfig(familyID, itemID, "addon", "Extra Seats "+itemID),
			},
			{
				// Dropping every field except id must produce no diff: the
				// computed attributes are reused from state via UseStateForUnknown.
				Config: testAccItemIDOnlyConfig(familyID, itemID, ""),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("chargebee_item.test", plancheck.ResourceActionNoop),
					},
				},
			},
		},
	})
}

func TestAccItemResourceArchiveByID(t *testing.T) {
	familyID := testAccRandomID("tf-acc-family")
	itemID := testAccRandomID("tf-acc-item")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccItemMeteredConfig(familyID, itemID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("chargebee_item.test", "metered", "true"),
					resource.TestCheckResourceAttr("chargebee_item.test", "status", "active"),
				),
			},
			{
				// Archiving a metered item with only id + status must be an
				// in-place update, never a replace, even though metered carries
				// RequiresReplace (UseStateForUnknown resolves it to the state value).
				Config: testAccItemIDOnlyConfig(familyID, itemID, "archived"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("chargebee_item.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("chargebee_item.test", "status", "archived"),
					resource.TestCheckResourceAttr("chargebee_item.test", "metered", "true"),
				),
			},
		},
	})
}

func TestAccItemResourceAdoptByIDNotFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "chargebee_item" "test" {
  id     = "tf-acc-never-created"
  status = "archived"
}
`,
				ExpectError: regexp.MustCompile(`no item with id "tf-acc-never-created" was found`),
			},
		},
	})
}

func TestAccItemResourcePartialIdentity(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "chargebee_item" "test" {
  id   = "tf-acc-partial"
  name = "Only a name"
}
`,
				ExpectError: regexp.MustCompile(`Set name, type, and item_family_id together`),
			},
		},
	})
}

func testAccItemConfig(familyID, itemID, itemType, name string) string {
	return fmt.Sprintf(`
resource "chargebee_item_family" "test" {
  id   = %[1]q
  name = "Item Test Family"
}

resource "chargebee_item" "test" {
  id             = %[2]q
  item_family_id = chargebee_item_family.test.id
  type           = %[3]q
  name           = %[4]q

  metadata = {
    managed_by = "terraform"
  }
}
`, familyID, itemID, itemType, name)
}

func testAccItemMeteredConfig(familyID, itemID string) string {
	return fmt.Sprintf(`
resource "chargebee_item_family" "test" {
  id   = %[1]q
  name = "Item Test Family"
}

resource "chargebee_item" "test" {
  id                = %[2]q
  item_family_id    = chargebee_item_family.test.id
  type              = "addon"
  name              = "Metered %[2]s"
  metered           = true
  usage_calculation = "sum_of_usages"
}
`, familyID, itemID)
}

// testAccItemIDOnlyConfig keeps the family resource alive but tracks the item by
// id alone (optionally forcing a status), so every other item attribute is read
// from Chargebee. status is omitted from the item block when empty.
func testAccItemIDOnlyConfig(familyID, itemID, status string) string {
	statusLine := ""
	if status != "" {
		statusLine = fmt.Sprintf("\n  status = %q", status)
	}
	return fmt.Sprintf(`
resource "chargebee_item_family" "test" {
  id   = %[1]q
  name = "Item Test Family"
}

resource "chargebee_item" "test" {
  id = %[2]q%[3]s
}
`, familyID, itemID, statusLine)
}
