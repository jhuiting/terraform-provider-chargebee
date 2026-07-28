# SPDX-License-Identifier: MPL-2.0

provider "chargebee" {
  site    = "acme-test"           # or env CHARGEBEE_SITE
  api_key = var.chargebee_api_key # or env CHARGEBEE_API_KEY
}
