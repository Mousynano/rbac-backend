# Architecture and development map

Status: Phase 1 bootstrap in progress, 2026-11-26. This document describes the code that exists today and proposes the target architecture. A proposed feature is not a delivered feature. The approved Phase 1 source of truth remains `.kiro/specs/phase-1-bootstrap/`; changes to its scope require a new approved spec and task list.

## Product boundaries

The platform owns accounts, authentication, diagram ownership, collaboration access, and persisted blueprint documents. A blueprint describes customer roles and permissions; those roles never authorize calls to this platform. The Go generator will eventually produce a standalone library for a customer's application, with identity and role assignment supplied by that application.

## Dependency direction

```text
cmd/api (composition root)
  -> HTTP adapters / middleware -> application use cases -> domain rules
  -> PostgreSQL, SMTP, Google OIDC, and realtime adapters

cmd/migrate, cmd/admin -> infrastructure and operator use cases
```

Dependencies are wired explicitly in `cmd`; domain rules must not import `net/http`, SQL, SMTP, OIDC, or a transport format. Use cases depend on small interfaces defined by their consumers only when they cross an I/O boundary. Adapters implement those interfaces. Prefer a concrete type when an interface would have only one use and add no testing or dependency benefit. Keep packages feature oriented, with no mandatory `controller/service/repository` hierarchy and no generic repository or DI framework. This follows the Google Go Style Guide's emphasis on simplicity, readable package names, `gofmt`, and small interfaces; Google does not prescribe a universal clean architecture directory template.

Current package map:

| Path | Responsibility | State |
| --- | --- | --- |
| `cmd/api` | Config, logger, dependency wiring, server lifecycle | Implemented for health only |
| `internal/config` | Parse and validate environment | Implemented, unit tests present |
| `internal/health` | Readiness use case, no HTTP dependency | Implemented; delegates DB and schema checks |
| `internal/platform/httpserver` | Server settings and HTTP health adapter | Implemented for health only |
| `internal/platform/database` | PostgreSQL pool and schema readiness adapter | Implemented |
| `cmd/migrate` | Explicit goose migration command | Implemented with initial SQL migrations |
| `cmd/admin` | Operator CLI | Stub |
| `internal/auth`, `internal/user`, `internal/blueprint`, `internal/diagram` | Planned feature modules | Absent |

Future feature modules should keep their model and use cases together, and put PostgreSQL/HTTP adapters in a feature subpackage when their size warrants it. An import cycle is a signal to move the interface to its consumer or split a responsibility, not to add a shared `utils` package.

## Data and API contracts

PostgreSQL is the authoritative store for users, identities, sessions, pending OTP/OIDC state, email policy, diagram metadata, blueprint JSONB, access grants, and event sequencing. IDs and ownership come from the server. Every diagram query filters by owner or an explicit collaborator grant. Role and permission identifiers in the blueprint have no relation to platform users.

Canonical blueprint JSON v1 is specified in `.kiro/specs/phase-1-bootstrap/design.md`: `schema_version`, `roles`, `permissions`, `grants`, `inheritance`, and optional `layout`. Validate strict shape, duplicate keys, references, and inheritance cycles before persistence. A diagram row has a separate monotonically increasing `version` for optimistic concurrency. `api/openapi.yaml` and `schemas/blueprint.v1.schema.json` are planned but absent. JSONB preserves semantic content, not object key order or whitespace.

“Save diagram in a JSON file” is proposed as an explicit import/export capability: download a versioned JSON document and upload it through the same validator and ownership checks. The database remains authoritative for collaborative editing; a local file is a portable snapshot, not a second live datastore. Define whether export includes only the blueprint or also diagram name and format metadata before freezing the contract. Never include session tokens, share tokens, owner email, or collaborator identities in an export.

## Authentication and email policy

The planned guard verifies an opaque session digest in PostgreSQL, expiry, account status, and email policy on each protected request; it obtains a trusted principal rather than reading owner IDs from JSON. Email/password and Google OIDC each require a pending email OTP challenge before a session is issued. Google uses authorization code, PKCE S256, state, nonce, and verified ID-token claims. Provider subjects are unique and linking to an existing account happens only after OTP. See the existing security steering and ADR-0003 for its accepted residual risk.

The requested “KYC” means email eligibility and abuse prevention here, not legal identity verification. A pinned disposable-domain snapshot, exact email/domain blocklist, syntax and DNS checks, and disabled accounts cover known temporary or spam addresses. They cannot prove that an email address is a real person's identity, and an unfamiliar legitimate domain must not be rejected solely for being uncommon. Operator updates should take effect on active sessions. Document list provenance, false-positive review, and an appeal/unblock path before production use.

## Collaboration proposal (new scope)

Add diagram collaborators with explicit roles such as viewer/editor; platform account identity is required for edits. An owner creates a revocable, high-entropy share link with expiry and a scoped capability. Decide whether link recipients can view anonymously or must sign in; editing should require a signed-in account and an explicit grant. Store only a hash of the link token. Resolve the link into a grant, and authorize every read/write/event against the current grant, including after revocation.

For realtime editing, start with an authoritative PostgreSQL version and WebSocket event stream. Each command carries `base_version` and an operation ID. The server checks access, applies a transaction, increments version, and broadcasts the committed event. A stale version returns a conflict and current snapshot/version; clients reconcile and retry. Presence and cursors can be ephemeral. Reconnect uses a cursor or a full snapshot if events have expired. Define retention, replay ordering, slow-client backpressure, revocation behavior, and multi-instance fan-out before implementation. PostgreSQL `LISTEN/NOTIFY` may wake instances, but durable events must be stored in a table; notification alone is not a replay log. Operational transforms or CRDTs are deferred until concurrent editing behavior demonstrates a need.

## Go code generation proposal (new scope)

Generate Go source from a validated, pinned blueprint snapshot. The generator should be pure: typed blueprint -> deterministic files, with no SQL or HTTP dependency. Emit a versioned manifest, library API, tests, and integration guide; format source with `go/format`, use stable ordering and golden tests, and compile generated fixtures in CI. Do not execute generated code on the server. Define the customer's identity/role assignment interface, inheritance semantics, package/module naming, output versioning, and compatibility policy in a separate spec before implementation. Any generator request against a collaborative diagram must name an exact committed version.

## Differences and current gaps

| Topic | Current reality / reason | Planned resolution |
| --- | --- | --- |
| Strict layered directories | Feature-oriented Go packages are easier to navigate than a global `domain/application/infrastructure` tree for this small monolith. | Preserve inward dependency direction and document each boundary. |
| HTTP health response and server | Health adapter shares `internal/platform/httpserver` with server settings because it is small. | Split into a dedicated transport package if HTTP routes grow. |
| Readiness | Checks DB ping and migration version 2. The check still needs an integration test against a fresh database. | Verify with Compose and schema upgrade tests. |
| Main API routes | Only liveness/readiness exist. | Implement middleware, auth, diagram routes in phases below. |
| Storage | Initial users/auth/diagrams schema exists; repositories and file export do not. | Implement feature adapters. |
| Security and collaboration | No guard, Google SSO, email blocklist, sharing, or realtime code exists. | Spec and implement; do not treat design prose as acceptance evidence. |
| Generator | No Go generator or generated output contract exists. Original SDD reserves generation for Phase 3. | New Phase 3 spec and acceptance gates. |
| Quality gate | Config and health HTTP tests exist. `go test ./...`, `go vet ./...`, and `go build ./cmd/...` pass outside the workspace sandbox. Compose bootstrap, migration v2, health/readiness, and Mailpit smoke checks pass. Integration, race, and vulnerability gates remain. | Run required gates as features land. |

## Requested feature coverage

| Requested capability | Current state | Delivery phase |
| --- | --- | --- |
| Auth guard / API middleware | Absent | 1 |
| Google SSO | Designed, absent in code | 1 |
| Email eligibility / blocklist (informal “KYC”) | Designed, absent in code | 1 |
| User database | Pool exists; user schema absent | 1 |
| Designer JSON structure | Blueprint v1 specified; validator/schema absent | 1 |
| Diagram persistence | Designed for PostgreSQL JSONB; absent | 1 |
| Share link and realtime collaboration | New scope, proposal above | 2 |
| Save / load diagram JSON file | New scope, proposal above | 3 |
| Go code generation | New scope, proposal above | 3 |

## Proposed development phases and gates

1. **Phase 0 — baseline and decisions.** Commit or otherwise preserve handover files, resolve toolchain, review existing version matrix, and approve amended requirements for collaboration/export/generation. Freeze API and threat model decisions that affect schema.
2. **Phase 1 — secure single-user core (existing M1–M4).** Finish Compose, migrations, schema-aware readiness, HTTP contract, auth guard, password/OTP, Google SSO, email eligibility/admin CLI, blueprint validator, and owner-scoped diagram CRUD. Gate: repeatable local setup; PostgreSQL/Mailpit integration and negative security tests; OpenAPI and schema match implementation.
3. **Phase 2 — sharing and collaboration (new).** Deliver scoped share links, collaborator grants, revoke/expiry, versioned edits, realtime stream, reconnect, and conflict handling. Gate: concurrent edits, cross-user denial, token leakage/revocation, multi-instance replay, and slow-client tests.
4. **Phase 3 — portable JSON and Go generator (new).** Add validated import/export; generate deterministic Go library, tests, and integration guide from an immutable version. Gate: schema round trip, golden fixtures, generated code compile/test, backward compatibility, no server-side execution.
5. **Phase 4 — hardening and release (extends existing M5).** CI gates, observability, backup/restore, retention, production SMTP/OIDC smoke tests with real infrastructure, load tests for collaborative fan-out, and security review. Gate: documented evidence and explicit production readiness decision.

Each phase needs its own requirements, design, tasks, and review gate. Update the original SDD to point to newly approved specs; do not silently mark its unchecked tasks complete.

## References

- [Google Go Style Guide](https://google.github.io/styleguide/go/guide)
- [Google Go Style Decisions](https://google.github.io/styleguide/go/decisions)
- [Google Go Best Practices](https://google.github.io/styleguide/go/best-practices)
- `.kiro/steering/` and `.kiro/specs/phase-1-bootstrap/`
