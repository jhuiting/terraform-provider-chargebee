# SPDX-License-Identifier: MPL-2.0

resource "chargebee_item" "standard" {
  id             = "standard"
  item_family_id = chargebee_item_family.main.id
  type           = "plan"
  name           = "Standard"
}

resource "chargebee_item" "api_calls" {
  id                = "api-calls"
  item_family_id    = chargebee_item_family.main.id
  type              = "addon"
  name              = "API calls"
  metered           = true
  usage_calculation = "sum_of_usages"
}
