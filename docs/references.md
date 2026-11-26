# Engineering references and version review
Reviewed documentation for starter pack on 2026-11-24. Version matrix locked after task 1 discovery and owner approval on 2026-11-24. See `docs/adr/` for decision records.

## Approved version matrix (locked 2026-11-24)

| Component | Version | License | Notes |
|---|---|---|---|
| Go toolchain / module directive | `go1.27.4` | BSD-3-Clause | 12 CVE fixes over 1.26; required by tech.md |
| PostgreSQL image | `postgres:18.7` | PostgreSQL License | 52 CVE fixes; released 2026-11-12 |
| `github.com/jackc/pgx/v5` | `v5.13.0` | MIT | Security fix GHSA-w9rp-vfrh-q8vv (panic on malformed response); released 2026-11-11 |
| `github.com/pressly/goose/v3` | `v3.30.1` | MIT | Released 2026-11-12; SQL-only migrations |
| `github.com/coreos/go-oidc/v3` | `v3.21.0` | Apache-2.0 | Nonce must be verified manually in app code |
| `golang.org/x/oauth2` | `v0.38.0` | BSD-3-Clause | PKCE S256 supported |
| `golang.org/x/crypto` | `v0.59.0` | BSD-3-Clause | Addresses GO-2026-5932; covers argon2 |
| `axllent/mailpit` (Docker) | `v1.34.0` | MIT | Security fix GHSA-v887-64wg-qqc9 (SMTP crash); released 2026-11-20 approx |
| disposable-email-domains | commit SHA TBD at task 5 | CC0 / public domain | Pin immutable commit + checksum; local snapshot only |

All versions are pinned. No `latest` tags in committed configs. Run `govulncheck ./...` after `go mod tidy` to confirm no open advisories.

## Kiro
- https://kiro.dev/docs/steering/ — workspace steering; YAML `inclusion: always` on all eight project rules so scope/security are not missed.
- https://kiro.dev/docs/specs/feature-specs/requirements-first/ — requirements, design and tasks with approval gates.
- https://kiro.dev/docs/specs/best-practices/ — spec files and context.
- https://kiro.dev/docs/getting-started/first-project/ — workspace flow.

## Go and data
- https://go.dev/doc/modules/layout — `cmd` and `internal` for server application.
- https://go.dev/doc/devel/release and https://go.dev/dl/ — Go 1.27 supported release line; verify patch at task 1.
- https://go.dev/doc/effective_go and https://go.dev/wiki/CodeReviewComments — idiomatic style (Effective Go is foundational, not comprehensive).
- https://pkg.go.dev/net/http — standard router and server.
- https://pkg.go.dev/github.com/jackc/pgx/v5/pgxpool — pgx v5 pool; observed v5.13.0 in docs, candidate subject to compatibility/security verification.
- https://github.com/pressly/goose — candidate migration tool; verify before adoption.
- https://www.postgresql.org/docs/18/datatype-json.html — PostgreSQL 18 JSONB; preserve semantic JSON rather than key order.
- https://www.postgresql.org/support/versioning/ — choose current patch of supported PostgreSQL 18.
- https://json-schema.org/draft/2020-12 — blueprint shape contract.
- https://go.dev/doc/tutorial/govulncheck — dependency vulnerability checks.

## Identity and security
- https://developers.google.com/identity/openid-connect/openid-connect — verify ID token and stable subject.
- https://developers.google.com/identity/openid-connect/reference — claim definitions.
- https://pkg.go.dev/github.com/coreos/go-oidc/v3/oidc — verifier API; explicitly check nonce and other claims not automatically enforced.
- https://pkg.go.dev/golang.org/x/oauth2 — OAuth flow and PKCE.
- https://pkg.go.dev/golang.org/x/crypto/argon2 — Argon2id.
- https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html — hash parameters.
- https://cheatsheetseries.owasp.org/cheatsheets/Multifactor_Authentication_Cheat_Sheet.html — OTP handling and limitations.
- https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html — cookie/session baseline.
- https://pages.nist.gov/800-63-4/sp800-63b.html — email OTP is not accepted as out-of-band authentication under this standard; do not claim NIST MFA compliance.
- https://github.com/disposable-email-domains/disposable-email-domains — candidate domain dataset; review license, pin commit and checksum, preserve attribution and sanity-check import. No per-request live lookup.

## SMTP
- https://mailpit.axllent.org/docs/install/docker/ — local SMTP capture and Docker ports.
- https://docker-mailserver.github.io/docker-mailserver/latest/ — self-hosted real SMTP candidate.
- https://docker-mailserver.github.io/docker-mailserver/latest/config/best-practices/dkim_dmarc_spf/ — DNS and authentication requirements.
- https://pkg.go.dev/net/smtp — frozen standard SMTP API; verify TLS, timeouts and authentication requirements.

## Relevant prior art for future discussion (not dependencies)
- https://casbin.org/docs/overview — policy/authorization library; useful contrast with our design-only Phase 1.
- https://www.openpolicyagent.org/docs/ — external policy evaluation model; not our runtime architecture.
Do not implement or integrate either in Phase 1.
