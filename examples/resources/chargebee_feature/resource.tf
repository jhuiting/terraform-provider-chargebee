# SPDX-License-Identifier: MPL-2.0

resource "chargebee_feature" "sso" {
  id   = "sso"
  name = "Single sign-on"
}

resource "chargebee_feature" "api_rate_limit" {
  id   = "api-rate-limit"
  name = "API rate limit"
  type = "quantity"
  unit = "requests/min"
}
