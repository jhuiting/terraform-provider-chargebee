# SPDX-License-Identifier: MPL-2.0

resource "chargebee_attached_item" "standard_api_calls" {
  parent_item_id = chargebee_item.standard.id
  item_id        = chargebee_item.api_calls.id
  type           = "recommended"
}
