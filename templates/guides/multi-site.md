---
page_title: "Managing multiple Chargebee sites"
description: |-
  Provision the same product catalog to several Chargebee sites with aliased
  providers and a shared catalog definition.
---

# Managing multiple Chargebee sites

Each Chargebee site has its own API host and API key, so each site gets its
own (aliased) provider block. The provider holds a client per instance;
aliases do not interfere with each other.

The `CHARGEBEE_SITE`/`CHARGEBEE_API_KEY` environment fallback is
single-valued — with more than one site, set `site` and `api_key` explicitly
per provider block.

## One aliased provider per site

Terraform requires provider blocks to be static, so declare one aliased
provider per site and select it on each resource with the `provider`
meta-argument.

```terraform
variable "chargebee_api_keys" {
  description = "Chargebee API key per site key."
  type        = map(string)
  sensitive   = true
}

provider "chargebee" {
  alias   = "eu"
  site    = "acme-eu"
  api_key = var.chargebee_api_keys["eu"]
}

provider "chargebee" {
  alias   = "us"
  site    = "acme-us"
  api_key = var.chargebee_api_keys["us"]
}

resource "chargebee_item_family" "main_eu" {
  provider = chargebee.eu
  id       = "main"
  name     = "Main"
}

resource "chargebee_item_family" "main_us" {
  provider = chargebee.us
  id       = "main"
  name     = "Main"
}
```

## Defining the catalog once

To keep a catalog defined once and apply it to every site, wrap the resources
in a local module of your own and call it once per site, passing a different
provider each time:

```terraform
module "catalog_eu" {
  source    = "./modules/catalog"
  providers = { chargebee = chargebee.eu }
}

module "catalog_us" {
  source    = "./modules/catalog"
  providers = { chargebee = chargebee.us }
}
```

Per-site deviations belong in module inputs, e.g. pass a `site_overrides`
variable and `merge` it inside the module.

## Supplying the API keys

Chargebee API keys are long-lived credentials. Do not keep production keys in
`.env` files or `*.tfvars` on developer machines.

- **CI (recommended):** store keys in the pipeline's secret store and inject
  them, e.g. `TF_VAR_chargebee_api_keys='{"eu":"...","us":"..."}'` from
  GitHub Actions secrets.
- **1Password:** `op run` to inject the variable at plan/apply time, or the
  `1password/onepassword` provider with a service account to read items as
  data sources.
- **Cloud secret managers:** read keys with an
  `aws_secretsmanager_secret_version` (or equivalent) data source and pass
  them into the provider blocks.

The `api_key` provider attribute is marked sensitive and is provider
configuration, so it is not written to state — but plans and state can
contain everything else; protect the backend accordingly.

## State layout

All sites in one root module (as above) gives one plan for the whole estate.
If sites must change independently — different release cadence, blast-radius
isolation — split per site into separate root modules or workspaces, each
with its own state.
