# ADR-0003 — Google OIDC automatic account linking after OTP

**Date:** 2026-11-24  
**Status:** Accepted  
**Deciders:** Product owner, architect  

## Context

When a user authenticates via Google OIDC with a `sub` (subject) that is not yet registered, but the verified `email` matches an existing platform account, two options exist:

1. Require a separate manual linking flow with explicit user confirmation.
2. Automatically link the Google subject to the existing account after the user completes email OTP.

Option 2 is simpler for the user and avoids building a linking UI in Phase 1.

## Decision

Automatic linking is approved as an MVP simplification for Phase 1. The rules are:

- Link **only** if: (a) the Google subject is new, (b) the email from the verified OIDC token matches a verified platform account, and (c) the user successfully completes OTP sent to that email address.
- Never replace or clear the existing password hash.
- Never reassign a subject already linked to a different account.
- Never merge account IDs.
- Unique constraints and a serialized transaction prevent race conditions.
- Record the link event in `auth_audit_logs`.

## Residual risk

Mailbox compromise or provider-level email reassignment could allow a third party to link their Google account to an existing user's platform account. This risk is explicitly accepted for Phase 1 and must be re-evaluated before any production deployment or Phase 2 work.

## Consequences

- No manual linking UI is built in Phase 1.
- Advanced MFA, device controls, and risk-based auth are deferred.
- The risk statement above must appear in the Phase 1 handover review document.
