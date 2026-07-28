// SPDX-License-Identifier: MPL-2.0

package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccItemEntitlementResource(t *testing.T) {
	familyID := testAccRandomID("tf-acc-family")
	planID := testAccRandomID("tf-acc-plan")
	featureID := testAccRandomID("tf-acc-feature")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccItemEntitlementConfig(familyID, planID, featureID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("chargebee_item_entitlement.test", "item_id", planID),
					resource.TestCheckResourceAttr("chargebee_item_entitlement.test", "item_type", "plan"),
					resource.TestCheckResourceAttr("chargebee_item_entitlement.test", "feature_id", featureID),
					resource.TestCheckResourceAttr("chargebee_item_entitlement.test", "value", "true"),
					resource.TestCheckResourceAttr("chargebee_item_entitlement.test", "id", planID+":"+featureID),
				),
			},
			{
				Config:   testAccItemEntitlementConfig(familyID, planID, featureID),
				PlanOnly: true,
			},
			{
				ResourceName:      "chargebee_item_entitlement.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: testAccItemEntitlementImportID("chargebee_item_entitlement.test"),
			},
		},
	})
}

func TestAccItemEntitlementResourceInvalidImportID(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "chargebee_item_entitlement" "test" {
  item_id    = "whatever"
  item_type  = "plan"
  feature_id = "whatever-feature"
}
`,
				ResourceName:  "chargebee_item_entitlement.test",
				ImportState:   true,
				ImportStateId: "no-colon",
				ExpectError:   regexp.MustCompile(`Invalid import ID`),
			},
		},
	})
}

func testAccItemEntitlementImportID(name string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return "", fmt.Errorf("resource %s not found in state", name)
		}
		return rs.Primary.Attributes["item_id"] + ":" + rs.Primary.Attributes["feature_id"], nil
	}
}

// testAccItemEntitlementConfig grants a fresh feature to a fresh plan, so the
// test is self-contained and leaves nothing behind on the test site.
func testAccItemEntitlementConfig(familyID, planID, featureID string) string {
	return fmt.Sprintf(`
resource "chargebee_item_family" "test" {
  id   = %[1]q
  name = "Item Entitlement Test Family"
}

resource "chargebee_item" "plan" {
  id             = %[2]q
  item_family_id = chargebee_item_family.test.id
  type           = "plan"
  name           = "Entitled Plan"
}

resource "chargebee_feature" "test" {
  id   = %[3]q
  name = %[3]q
}

resource "chargebee_item_entitlement" "test" {
  item_id    = chargebee_item.plan.id
  item_type  = chargebee_item.plan.type
  feature_id = chargebee_feature.test.id
}
`, familyID, planID, featureID)
}
