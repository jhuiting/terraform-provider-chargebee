// SPDX-License-Identifier: MPL-2.0

package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccItemPriceResourcePerUnit(t *testing.T) {
	familyID := testAccRandomID("tf-acc-family")
	itemID := testAccRandomID("tf-acc-item")
	priceID := testAccRandomID("tf-acc-price")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccItemPriceBase(familyID, itemID) + fmt.Sprintf(`
resource "chargebee_item_price" "test" {
  id            = %[1]q
  item_id       = chargebee_item.test.id
  name          = %[1]q
  currency_code = "EUR"
  pricing_model = "per_unit"
  price         = 4900
  period        = 1
  period_unit   = "month"
}
`, priceID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("chargebee_item_price.test", "price", "4900"),
					resource.TestCheckResourceAttr("chargebee_item_price.test", "status", "active"),
					resource.TestCheckResourceAttr("chargebee_item_price.test", "free_quantity", "0"),
				),
			},
			{
				// In-place price update.
				Config: testAccItemPriceBase(familyID, itemID) + fmt.Sprintf(`
resource "chargebee_item_price" "test" {
  id            = %[1]q
  item_id       = chargebee_item.test.id
  name          = %[1]q
  currency_code = "EUR"
  pricing_model = "per_unit"
  price         = 5900
  period        = 1
  period_unit   = "month"
}
`, priceID),
				Check: resource.TestCheckResourceAttr("chargebee_item_price.test", "price", "5900"),
			},
			{
				ResourceName:      "chargebee_item_price.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccItemPriceResourceTiered(t *testing.T) {
	familyID := testAccRandomID("tf-acc-family")
	itemID := testAccRandomID("tf-acc-item")
	priceID := testAccRandomID("tf-acc-price")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccItemPriceBase(familyID, itemID) + fmt.Sprintf(`
resource "chargebee_item_price" "test" {
  id            = %[1]q
  item_id       = chargebee_item.test.id
  name          = %[1]q
  currency_code = "EUR"
  pricing_model = "tiered"
  period        = 1
  period_unit   = "month"

  tiers = [
    { starting_unit = 1, ending_unit = 10, price = 1000 },
    { starting_unit = 11, price = 800 },
  ]
}
`, priceID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("chargebee_item_price.test", "tiers.#", "2"),
					resource.TestCheckResourceAttr("chargebee_item_price.test", "tiers.0.price", "1000"),
					resource.TestCheckNoResourceAttr("chargebee_item_price.test", "tiers.1.ending_unit"),
				),
			},
			{
				ResourceName:      "chargebee_item_price.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// Note: multiple currencies require the currencies to be enabled on the
// Chargebee site; test sites often have just one. Two price points on one
// item exercise the same fan-out mechanics.
func TestAccItemPriceResourceMultiplePricePoints(t *testing.T) {
	familyID := testAccRandomID("tf-acc-family")
	itemID := testAccRandomID("tf-acc-item")
	priceID := testAccRandomID("tf-acc-price")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccItemPriceBase(familyID, itemID) + fmt.Sprintf(`
resource "chargebee_item_price" "month" {
  id            = "%[1]s-eur-month"
  item_id       = chargebee_item.test.id
  name          = "%[1]s-eur-month"
  currency_code = "EUR"
  pricing_model = "flat_fee"
  price         = 1000
  period        = 1
  period_unit   = "month"
}

resource "chargebee_item_price" "year" {
  id            = "%[1]s-eur-year"
  item_id       = chargebee_item.test.id
  name          = "%[1]s-eur-year"
  currency_code = "EUR"
  pricing_model = "flat_fee"
  price         = 10000
  period        = 1
  period_unit   = "year"
}
`, priceID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("chargebee_item_price.month", "period_unit", "month"),
					resource.TestCheckResourceAttr("chargebee_item_price.year", "period_unit", "year"),
				),
			},
		},
	})
}

func TestAccItemPriceResourceAdoptByIDNoOp(t *testing.T) {
	familyID := testAccRandomID("tf-acc-family")
	itemID := testAccRandomID("tf-acc-item")
	priceID := testAccRandomID("tf-acc-price")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccItemPricePerUnitConfig(familyID, itemID, priceID),
			},
			{
				// Dropping every field except id must produce no diff.
				Config: testAccItemPriceIDOnlyConfig(familyID, itemID, priceID, ""),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("chargebee_item_price.test", plancheck.ResourceActionNoop),
					},
				},
			},
		},
	})
}

func TestAccItemPriceResourceArchiveByID(t *testing.T) {
	familyID := testAccRandomID("tf-acc-family")
	itemID := testAccRandomID("tf-acc-item")
	priceID := testAccRandomID("tf-acc-price")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccItemPricePerUnitConfig(familyID, itemID, priceID),
				Check:  resource.TestCheckResourceAttr("chargebee_item_price.test", "status", "active"),
			},
			{
				// Archiving with only id + status must be an in-place update,
				// never a replace (currency_code carries RequiresReplace but is
				// resolved to the state value by UseStateForUnknown).
				Config: testAccItemPriceIDOnlyConfig(familyID, itemID, priceID, "archived"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("chargebee_item_price.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("chargebee_item_price.test", "status", "archived"),
					resource.TestCheckResourceAttr("chargebee_item_price.test", "price", "4900"),
				),
			},
		},
	})
}

func TestAccItemPriceResourceAdoptByIDNotFound(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "chargebee_item_price" "test" {
  id     = "tf-acc-price-never-created"
  status = "archived"
}
`,
				ExpectError: regexp.MustCompile(`no price with id "tf-acc-price-never-created" was found`),
			},
		},
	})
}

func TestAccItemPriceResourcePartialIdentity(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "chargebee_item_price" "test" {
  id   = "tf-acc-price-partial"
  name = "Only a name"
}
`,
				ExpectError: regexp.MustCompile(`Set item_id, name, currency_code, and pricing_model together`),
			},
		},
	})
}

func testAccItemPricePerUnitConfig(familyID, itemID, priceID string) string {
	return testAccItemPriceBase(familyID, itemID) + fmt.Sprintf(`
resource "chargebee_item_price" "test" {
  id            = %[1]q
  item_id       = chargebee_item.test.id
  name          = %[1]q
  currency_code = "EUR"
  pricing_model = "per_unit"
  price         = 4900
  period        = 1
  period_unit   = "month"
}
`, priceID)
}

// testAccItemPriceIDOnlyConfig keeps the family and item alive but tracks the
// price by id alone (optionally forcing a status), so every other price
// attribute is read from Chargebee.
func testAccItemPriceIDOnlyConfig(familyID, itemID, priceID, status string) string {
	statusLine := ""
	if status != "" {
		statusLine = fmt.Sprintf("\n  status = %q", status)
	}
	return testAccItemPriceBase(familyID, itemID) + fmt.Sprintf(`
resource "chargebee_item_price" "test" {
  id = %[1]q%[2]s
}
`, priceID, statusLine)
}

func testAccItemPriceBase(familyID, itemID string) string {
	return fmt.Sprintf(`
resource "chargebee_item_family" "test" {
  id   = %[1]q
  name = "Price Test Family"
}

resource "chargebee_item" "test" {
  id             = %[2]q
  item_family_id = chargebee_item_family.test.id
  type           = "addon"
  name           = %[2]q
}
`, familyID, itemID)
}
