// SPDX-License-Identifier: MPL-2.0

package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccItemFamilyResource(t *testing.T) {
	id := testAccRandomID("tf-acc-family")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccItemFamilyConfig(id, "Acceptance Test Family", "created by acceptance test"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("chargebee_item_family.test", "id", id),
					resource.TestCheckResourceAttr("chargebee_item_family.test", "name", "Acceptance Test Family"),
					resource.TestCheckResourceAttr("chargebee_item_family.test", "status", "active"),
				),
			},
			{
				Config: testAccItemFamilyConfig(id, "Renamed Family", "created by acceptance test"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("chargebee_item_family.test", "name", "Renamed Family"),
				),
			},
			{
				ResourceName:      "chargebee_item_family.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccItemFamilyDataSource(t *testing.T) {
	id := testAccRandomID("tf-acc-family-ds")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccItemFamilyConfig(id, "Data Source Family", "lookup target") + `
data "chargebee_item_family" "test" {
  id = chargebee_item_family.test.id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.chargebee_item_family.test", "name", "Data Source Family"),
					resource.TestCheckResourceAttr("data.chargebee_item_family.test", "status", "active"),
				),
			},
		},
	})
}

func testAccItemFamilyConfig(id, name, description string) string {
	return fmt.Sprintf(`
resource "chargebee_item_family" "test" {
  id          = %[1]q
  name        = %[2]q
  description = %[3]q
}
`, id, name, description)
}
