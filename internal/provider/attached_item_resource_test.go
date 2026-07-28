// SPDX-License-Identifier: MPL-2.0

package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccAttachedItemResource(t *testing.T) {
	familyID := testAccRandomID("tf-acc-family")
	planID := testAccRandomID("tf-acc-plan")
	chargeID := testAccRandomID("tf-acc-charge")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAttachedItemConfig(familyID, planID, chargeID, "subscription_creation"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("chargebee_attached_item.test", "parent_item_id", planID),
					resource.TestCheckResourceAttr("chargebee_attached_item.test", "item_id", chargeID),
					resource.TestCheckResourceAttr("chargebee_attached_item.test", "charge_on_event", "subscription_creation"),
					resource.TestCheckResourceAttrSet("chargebee_attached_item.test", "id"),
				),
			},
			{
				Config:   testAccAttachedItemConfig(familyID, planID, chargeID, "subscription_creation"),
				PlanOnly: true,
			},
			{
				// Changing the parent must force replacement.
				Config: testAccAttachedItemConfig(familyID, planID+"-alt", chargeID, "subscription_creation"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("chargebee_attached_item.test", plancheck.ResourceActionReplace),
					},
				},
			},
			{
				ResourceName:      "chargebee_attached_item.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: testAccAttachedItemImportID("chargebee_attached_item.test"),
			},
		},
	})
}

func TestAccAttachedItemResourceInvalidChargeOnEvent(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "chargebee_attached_item" "test" {
  parent_item_id  = "whatever"
  item_id         = "whatever-child"
  charge_on_event = "never_valid"
}
`,
				ExpectError: regexp.MustCompile(`charge_on_event`),
			},
		},
	})
}

func testAccAttachedItemImportID(name string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return "", fmt.Errorf("resource %s not found in state", name)
		}
		return rs.Primary.Attributes["parent_item_id"] + ":" + rs.Primary.Attributes["id"], nil
	}
}

func testAccAttachedItemConfig(familyID, planID, chargeID, chargeOnEvent string) string {
	return fmt.Sprintf(`
resource "chargebee_item_family" "test" {
  id   = %[1]q
  name = "Attached Item Test Family"
}

resource "chargebee_item" "plan" {
  id             = %[2]q
  item_family_id = chargebee_item_family.test.id
  type           = "plan"
  name           = "Parent Plan"
}

resource "chargebee_item" "charge" {
  id             = %[3]q
  item_family_id = chargebee_item_family.test.id
  type           = "charge"
  name           = "Child Charge"
}

resource "chargebee_attached_item" "test" {
  parent_item_id  = chargebee_item.plan.id
  item_id         = chargebee_item.charge.id
  charge_on_event = %[4]q
}
`, familyID, planID, chargeID, chargeOnEvent)
}
