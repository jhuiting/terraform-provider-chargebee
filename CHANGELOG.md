# Changelog

## v1.0.0 (unreleased)

Initial stable release. Schemas are covered by the stability policy
documented on the provider index page: breaking changes only in major
versions, deprecations announced at least six months before removal.

FEATURES:

- **New resource:** `chargebee_item_family`
- **New resource:** `chargebee_item` (plan / addon / charge, metered items)
- **New resource:** `chargebee_item_price` (flat_fee, per_unit, tiered,
  volume, stairstep; multi-currency; trials; billing cycles)
- **New resource:** `chargebee_attached_item` — attaches a child item
  (addon / charge) to a parent plan item
- **New resource:** `chargebee_feature` — entitlement feature, activated
  automatically on create
- **New resource:** `chargebee_item_entitlement` — grants a feature to an
  item (plan / addon / charge)
- **New data source:** `chargebee_item_family`

NOTES:

- Destroy attempts a real delete and falls back to archiving (with a
  warning) when Chargebee refuses because the object is in use. Item
  families cannot be archived and are left in place with a warning.
- Chargebee cannot attach cancellation to read/delete API calls (SDK
  limitation); a 60s HTTP timeout bounds them instead.

<!--
Template for future releases:

## vX.Y.Z (unreleased)

BACKWARDS INCOMPATIBILITIES:
FEATURES:
ENHANCEMENTS:
BUG FIXES:
NOTES:

Reference PRs/issues as (#123).
-->
