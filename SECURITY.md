# Security Policy

## Reporting a vulnerability

Please do not report security vulnerabilities through public GitHub issues.

Report them privately via GitHub's **Report a vulnerability** button
(Security tab) on this repository. You should receive a response within
5 business days.

Please include reproduction steps and the provider version. Credentials in
crash output: note that Terraform may log request details at `TF_LOG=debug`;
never attach debug logs without redacting your API key.

## Supported versions

Only the latest minor release receives security fixes.

## Scope notes

- The provider sends your Chargebee API key only to
  `https://<site>.chargebee.com` over TLS.
- The API key is marked sensitive in the provider schema and is not written
  to Terraform state; plan files and state can still contain catalog data —
  protect your state backend.
