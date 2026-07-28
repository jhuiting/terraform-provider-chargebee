# Contributing

Thanks for your interest in improving the Chargebee Terraform provider.

Security vulnerabilities: follow [SECURITY.md](SECURITY.md) — never a public
issue. All participation is covered by the
[Code of Conduct](CODE_OF_CONDUCT.md).

## Development setup

Requirements: Go ≥ 1.23, Terraform ≥ 1.9, `golangci-lint`, `goimports`.

```sh
make verify   # fmt + lint + build + unit tests — must be green before a PR
```

Build a local provider binary and point Terraform at it via
`dev_overrides` (see the [README](README.md#local-development)).

## Testing

| Tier | Command | Credentials |
|---|---|---|
| Unit | `make test` | none |
| Acceptance | `make testacc` | Chargebee **TEST** site |

Acceptance tests create and destroy real catalog objects. Put test-site
credentials in a gitignored `.env` (`CHARGEBEE_SITE`, `CHARGEBEE_API_KEY`);
the Makefile picks it up automatically. Never use a production site: archived
ids stay reserved forever.

## Pull requests

- One logical change per PR; include tests for behavior changes.
- New resource attributes need: schema + plan modifiers, adapter mapping in
  `internal/chargebee/client.go`, acceptance test coverage, and regenerated
  docs (`make docs` — edit `templates/`, never `docs/` directly).
- Follow the existing error-handling pattern: SDK errors are normalized to
  `chargebee.APIError` in the adapter; the provider layer only uses
  `IsNotFound`/`IsInUse`.
- Commit messages: [Conventional Commits](https://www.conventionalcommits.org/)
  (`feat:`, `fix:`, `docs:`, `chore:`).

## Releasing (maintainers)

Tag `vX.Y.Z` on `main`; the release workflow builds and signs artifacts with
goreleaser for the Terraform Registry.
