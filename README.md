# terraform-provider-chargebee

Terraform provider for the [Chargebee](https://www.chargebee.com/) Product
Catalog 2.0 — manage product families, items (plans, add-ons, charges), item
prices, and attached items as Terraform resources.

> **Note:** this is a community provider, not an official Chargebee product.
> It manages Product Catalog **2.0** objects only (item families, items, item
> prices) — not PC 1.0 plans/addons, subscriptions, or customers.

## Requirements

| Name | Version |
|---|---|
| Terraform | >= 1.9 |
| Go (development only) | >= 1.23 |
| Chargebee | Product Catalog 2.0 site |

## Quick start

```hcl
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

## Resources

| Name | Manages |
|---|---|
| `chargebee_item_family` | Product families |
| `chargebee_item` | Items (type `plan`, `addon`, or `charge`) |
| `chargebee_item_price` | Item prices (per currency/period, all pricing models) |
| `chargebee_attached_item` | Attachments linking a child item to a parent plan |
| `data.chargebee_item_family` | Lookup of an existing product family |

## Provider configuration

```hcl
provider "chargebee" {
  site    = "acme-test"            # or env CHARGEBEE_SITE
  api_key = var.chargebee_api_key  # or env CHARGEBEE_API_KEY (sensitive)
}
```

## Destroy semantics

Chargebee refuses to delete catalog objects that are (or were) in use by
subscriptions. On `terraform destroy` this provider first attempts a real
delete; if Chargebee refuses, it archives the object instead, emits a warning,
and removes it from state. Item families have no archived status — they are
left in place with a warning. Note that archived ids stay reserved in Chargebee.

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
