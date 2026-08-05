# terraform-provider-chargebee

[![Terraform Registry](https://img.shields.io/badge/Terraform%20Registry-jhuiting%2Fchargebee-844FBA?logo=terraform)](https://registry.terraform.io/providers/jhuiting/chargebee/latest)
[![Release](https://img.shields.io/github/v/release/jhuiting/terraform-provider-chargebee?logo=github&sort=semver)](https://github.com/jhuiting/terraform-provider-chargebee/releases/latest)
[![Test](https://github.com/jhuiting/terraform-provider-chargebee/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/jhuiting/terraform-provider-chargebee/actions/workflows/test.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/jhuiting/terraform-provider-chargebee)](https://goreportcard.com/report/github.com/jhuiting/terraform-provider-chargebee)
[![Go Version](https://img.shields.io/github/go-mod/go-version/jhuiting/terraform-provider-chargebee?logo=go)](go.mod)
[![License: MPL-2.0](https://img.shields.io/badge/License-MPL--2.0-blue.svg)](LICENSE)

Terraform provider for the [Chargebee](https://www.chargebee.com/) Product
Catalog 2.0 — manage product families, items (plans, add-ons, charges), item
prices, features, and entitlements as Terraform resources.

**Published on the Terraform Registry:
[registry.terraform.io/providers/jhuiting/chargebee](https://registry.terraform.io/providers/jhuiting/chargebee/latest).**
Full resource and data-source reference lives in the
[registry documentation](https://registry.terraform.io/providers/jhuiting/chargebee/latest/docs).

> **Note:** this is a community provider, not an official Chargebee product.
> It manages Product Catalog **2.0** objects only (item families, items, item
> prices, features, entitlements) — not PC 1.0 plans/addons, subscriptions, or
> customers.

## Requirements

| Name | Version |
|---|---|
| Terraform | >= 1.9 |
| Go (development only) | >= 1.23 |
| Chargebee | Product Catalog 2.0 site |

## Quick start

```hcl
terraform {
  required_providers {
    chargebee = {
      source  = "jhuiting/chargebee"
      version = "~> 0.1"
    }
  }
}

provider "chargebee" {} # CHARGEBEE_SITE + CHARGEBEE_API_KEY from env

resource "chargebee_item_family" "main" {
  id   = "main"
  name = "Main"
}

resource "chargebee_item" "standard" {
  id             = "standard"
  item_family_id = chargebee_item_family.main.id
  type           = "plan"
  name           = "Standard"
}

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
```

## Resources & data sources

| Name | Manages | Example | Docs |
|---|---|---|---|
| `chargebee_item_family` | Product families | [example](examples/resources/chargebee_item_family/resource.tf) | [docs](docs/resources/item_family.md) |
| `chargebee_item` | Items (type `plan`, `addon`, or `charge`) | [example](examples/resources/chargebee_item/resource.tf) | [docs](docs/resources/item.md) |
| `chargebee_item_price` | Item prices (per currency/period, all pricing models) | [example](examples/resources/chargebee_item_price/resource.tf) | [docs](docs/resources/item_price.md) |
| `chargebee_attached_item` | Attachments linking a child item to a parent plan | [example](examples/resources/chargebee_attached_item/resource.tf) | [docs](docs/resources/attached_item.md) |
| `chargebee_feature` | Entitlement features (switch, quantity, range, custom) | [example](examples/resources/chargebee_feature/resource.tf) | [docs](docs/resources/feature.md) |
| `chargebee_item_entitlement` | Grants a feature to an item | [example](examples/resources/chargebee_item_entitlement/resource.tf) | [docs](docs/resources/item_entitlement.md) |
| `data.chargebee_item_family` | Lookup of an existing product family | [example](examples/data-sources/chargebee_item_family/data-source.tf) | [docs](docs/data-sources/item_family.md) |

Provider configuration example:
[examples/provider/provider.tf](examples/provider/provider.tf). Guide for
running against several sites: [docs/guides/multi-site.md](docs/guides/multi-site.md).

## Provider configuration

```hcl
provider "chargebee" {
  site    = "acme-test"            # or env CHARGEBEE_SITE
  api_key = var.chargebee_api_key  # or env CHARGEBEE_API_KEY (sensitive)
}
```

## Entitlements

Entitlements gate what an item grants its subscribers. Two resources model them,
and they work together:

- **`chargebee_feature`** defines a capability once at the site level. A feature
  has a `type` that controls how its value is interpreted:
  - `switch` (the default) — a boolean capability, granted with value `true`.
  - `quantity` — a numeric allowance (e.g. `600` API calls); pair with a `unit`.
  - `range` — a numeric value within a range; pair with a `unit`.
  - `custom` — a free-form value you interpret yourself.

  Features created by this provider are activated automatically so items can be
  granted them immediately.

- **`chargebee_item_entitlement`** grants one feature to one item and sets the
  `value` for that item. The value's meaning follows the feature's `type` — a
  switch entitlement uses `true`, a quantity feature carries the numeric
  allowance. The entitlement's `item_type` must match the item's `type`.

Chargebee's entitlement upsert is idempotent: an entitlement already present in
Chargebee is adopted into state and its value reconciled to your configuration
on the first apply, rather than erroring on a conflict.

```hcl
# Define features once.
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

# Grant them to a plan, with per-item values.
resource "chargebee_item_entitlement" "standard_sso" {
  item_id    = chargebee_item.standard.id
  item_type  = chargebee_item.standard.type
  feature_id = chargebee_feature.sso.id
}

resource "chargebee_item_entitlement" "standard_rate_limit" {
  item_id    = chargebee_item.standard.id
  item_type  = chargebee_item.standard.type
  feature_id = chargebee_feature.api_rate_limit.id
  value      = "600"
}
```

Full examples:
[feature](examples/resources/chargebee_feature/resource.tf),
[item entitlement](examples/resources/chargebee_item_entitlement/resource.tf).

## Destroy semantics

Chargebee refuses to delete catalog objects that are (or were) in use by
subscriptions or invoices. On `terraform destroy` this provider:

1. Attempts a real delete.
2. If Chargebee refuses, archives the object instead (items, item prices, and
   features), emits a warning, and removes it from state. Archived ids stay
   reserved in Chargebee.
3. Item families cannot be archived: an in-use family is left in place with a
   warning and removed from state.
4. Item entitlements are revoked directly; there is no archived state for them.

## Local development

Build and install the provider into `GOBIN`:

```sh
make install
```

Point Terraform at the local build via `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "registry.terraform.io/jhuiting/chargebee" = "/Users/<you>/go/bin"
  }
  direct {}
}
```

With `dev_overrides` active, skip `terraform init` and run `terraform plan`
/ `apply` directly.

### Testing

Two tiers; only the last needs credentials:

1. **Unit tests** — `make test`. No credentials; covers error classification,
   delete fallback, price validators, and the SDK adapter against a fake
   HTTP server.
2. **Acceptance tests** — `make testacc`. Runs against a real Chargebee
   **TEST** site (creates and destroys real catalog objects). Put the test
   site's credentials in a gitignored `.env` in the repo root:

   ```sh
   # .env — TEST site only, never production keys
   CHARGEBEE_SITE=your-test-site
   CHARGEBEE_API_KEY=your-test-api-key
   ```

   The Makefile includes `.env` automatically — plain `KEY=value` lines, no
   quotes, no `export`:

   ```sh
   make testacc
   ```

   For raw `terraform` commands in a config of your own, source it once:

   ```sh
   set -a; source .env; set +a
   terraform plan
   ```

   Production keys never belong in `.env` files — see the multi-site guide
   (`docs/guides/multi-site.md`) for supplying keys in CI.

### Logging

The provider logs via `terraform-plugin-log` (`tflog`) instead of `log/slog`:
provider log output must flow through Terraform's structured logging system
(`TF_LOG`), which `tflog` is the supported API for.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Bug reports and PRs welcome.

## License

[Mozilla Public License 2.0](LICENSE).
