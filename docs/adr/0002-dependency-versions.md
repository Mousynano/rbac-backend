# ADR-0002 — Approved dependency and container version matrix

**Date:** 2026-11-24  
**Status:** Accepted  
**Deciders:** Product owner, architect  

## Context

Before writing any application code the engineering team performed a task-1 discovery review: inspecting the workspace, checking official release sources, reviewing security advisories, and confirming license compatibility. The steering (`tech.md`) prohibits floating `latest` tags and unapproved additions.

Two versions proposed during discovery were corrected by the product owner:

- `pgx v5.11.0` → **`v5.13.0`**: v5.13.0 (2026-11-11) includes a security fix (GHSA-w9rp-vfrh-q8vv) for a panic on malformed server responses that causes an index-out-of-range in `Rows.Values`. It also adds automatic TLS client certificate reloading and `database/sql` temporal precision metadata.
- `mailpit v1.33.0` → **`v1.34.0`**: v1.34.0 fixes a security vulnerability (GHSA-v887-64wg-qqc9) where a specially crafted SMTP message could crash the server. The release notes explicitly state updating is recommended. The release also includes extensive SMTP and HTML-parser hardening.

## Decision

The following versions are locked for Phase 1. No dependency may be changed without a new ADR and owner approval.

| Component | Version | License | Security notes |
|---|---|---|---|
| Go toolchain | `go1.27.4` | BSD-3-Clause | See ADR-0001 |
| PostgreSQL image | `postgres:18.7` | PostgreSQL License | 52 CVEs fixed over 18.6 |
| `github.com/jackc/pgx/v5` | `v5.13.0` | MIT | GHSA-w9rp-vfrh-q8vv fixed |
| `github.com/pressly/goose/v3` | `v3.30.1` | MIT | Active maintenance; SQL-only |
| `github.com/coreos/go-oidc/v3` | `v3.21.0` | Apache-2.0 | Nonce verified manually in app |
| `golang.org/x/oauth2` | `v0.38.0` | BSD-3-Clause | PKCE S256 supported |
| `golang.org/x/crypto` | `v0.59.0` | BSD-3-Clause | GO-2026-5932 addressed |
| `axllent/mailpit` (Docker) | `v1.34.0` | MIT | GHSA-v887-64wg-qqc9 fixed |
| disposable-email-domains | commit SHA TBD (task 5) | CC0 / public domain | Pin immutable commit + file checksum |

All are direct dependencies only. Transitive dependencies are resolved by `go mod tidy` and recorded in `go.sum`. No prerelease or pseudo-version tags are used.

## Consequences

- `go.sum` is committed and must not be modified manually.
- The disposable-email-domains commit is deferred to task 5; the pinned SHA and SHA-256 file checksum must be recorded in this ADR when resolved.
- When any dependency receives a security patch, open a new ADR revision, obtain owner approval, and update `go.mod` and this record together.
- `govulncheck ./...` is a required gate in every CI run; failures must be resolved or owner-approved exceptions documented here.
