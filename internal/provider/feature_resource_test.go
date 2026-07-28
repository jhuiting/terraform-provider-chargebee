// SPDX-License-Identifier: MPL-2.0

package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccFeatureResource(t *testing.T) {
	featureID := testAccRandomID("tf-acc-feature")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccFeatureConfig(featureID, featureID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("chargebee_feature.test", "id", featureID),
					resource.TestCheckResourceAttr("chargebee_feature.test", "name", featureID),
					resource.TestCheckResourceAttr("chargebee_feature.test", "type", "switch"),
					resource.TestCheckResourceAttr("chargebee_feature.test", "status", "active"),
				),
			},
			{
				Config:   testAccFeatureConfig(featureID, featureID),
				PlanOnly: true,
			},
			{
				Config: testAccFeatureConfig(featureID, featureID+"-renamed"),
				Check:  resource.TestCheckResourceAttr("chargebee_feature.test", "name", featureID+"-renamed"),
			},
			{
				ResourceName:      "chargebee_feature.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccFeatureResourceInvalidType(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "chargebee_feature" "test" {
  id   = "whatever"
  name = "Whatever"
  type = "not_a_type"
}
`,
				ExpectError: regexp.MustCompile(`type`),
			},
		},
	})
}

func TestAccFeatureResourceUnitOnSwitch(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "chargebee_feature" "test" {
  id   = "whatever"
  name = "Whatever"
  unit = "seats"
}
`,
				ExpectError: regexp.MustCompile(`unit requires a quantity or range feature`),
			},
		},
	})
}

func testAccFeatureConfig(id, name string) string {
	return fmt.Sprintf(`
resource "chargebee_feature" "test" {
  id   = %[1]q
  name = %[2]q
}
`, id, name)
}
