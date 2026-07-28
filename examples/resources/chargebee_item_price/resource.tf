# SPDX-License-Identifier: MPL-2.0

resource "chargebee_item_price" "standard_eur" {
  id            = "standard-eur"
  item_id       = chargebee_item.standard.id
  name          = "standard-eur"
  currency_code = "EUR"
  pricing_model = "per_unit"
  price         = 4900 # cents
  period        = 1
  period_unit   = "month"
}

resource "chargebee_item_price" "api_calls_eur" {
  id            = "api-calls-eur"
  item_id       = chargebee_item.api_calls.id
  name          = "api-calls-eur"
  currency_code = "EUR"
  pricing_model = "tiered"
  period        = 1
  period_unit   = "month"

  tiers = [
    { starting_unit = 1, ending_unit = 100000, price = 0 },
    { starting_unit = 100001, price = 1 },
  ]
}
