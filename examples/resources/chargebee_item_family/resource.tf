# SPDX-License-Identifier: MPL-2.0

resource "chargebee_item_family" "main" {
  id          = "main"
  name        = "Main"
  description = "Primary product family"
}
