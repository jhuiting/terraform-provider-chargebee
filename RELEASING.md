# Releasing

One-time setup and per-release steps for publishing to the Terraform Registry.

## One-time setup

The repository is named `terraform-provider-chargebee`; the registry only
detects repositories named `terraform-provider-{name}`, and the provider
address is `jhuiting/chargebee`.

1. **Generate a GPG signing key** (RSA — the registry does not accept ECC):
   ```sh
   gpg --full-generate-key        # RSA and RSA, 4096 bits
   gpg --armor --export-secret-keys <key-id>   # → GPG_PRIVATE_KEY repo secret
   gpg --armor --export <key-id>               # → registry public key
   ```
   Add repository secrets `GPG_PRIVATE_KEY` and `PASSPHRASE`.
2. **Register the public key** at
   https://registry.terraform.io/settings/gpg-keys under the
   `jhuiting` namespace (sign in with the GitHub account that owns the
   repository).
3. **Publish the provider**: registry → Publish → Provider → select the repo.
   Requires at least one release (step below) to exist.

## Per release

1. Update `CHANGELOG.md`: move entries from the unreleased section under the
   version being released (sections: BACKWARDS INCOMPATIBILITIES / FEATURES /
   ENHANCEMENTS / BUG FIXES / NOTES; reference PRs as `(#123)`).
2. Verify locally: `make verify && make testacc` (TEST site).
3. Tag and push:
   ```sh
   git tag v1.0.0
   git push origin v1.0.0
   ```
   The release workflow builds the multi-platform matrix, signs the checksums,
   and publishes the GitHub release; the registry ingests it automatically
   (webhook after first manual publish).
4. Check the release run and the new version on the registry page.

## Version policy

Semantic versioning. Schemas are stable within a major version; deprecations
are announced in the changelog at least six months before removal (see the
Stability section in `docs/index.md`).
