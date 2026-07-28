# SPDX-License-Identifier: MPL-2.0

# Switch feature: granted with the default value "true".
resource "chargebee_item_entitlement" "standard_sso" {
  item_id    = chargebee_item.standard.id
  item_type  = chargebee_item.standard.type
  feature_id = chargebee_feature.sso.id
}

# Quantity feature: the value is the granted amount.
resource "chargebee_item_entitlement" "standard_rate_limit" {
  item_id    = chargebee_item.standard.id
  item_type  = chargebee_item.standard.type
  feature_id = chargebee_feature.api_rate_limit.id
  value      = "600"
}
