# ADR-0005 — PostgreSQL as sole persistent store; no external cache

**Date:** 2026-11-24  
**Status:** Accepted  
**Deciders:** Product owner, architect  

## Context

Distributed caches (Redis, Memcached) are commonly used for rate limiting, session management, and OTP state. They add operational complexity, require additional infrastructure, and introduce consistency challenges.

## Decision

PostgreSQL is the only persistent store for Phase 1. Rate limits use atomic `INSERT ON CONFLICT UPDATE` fixed-window counters. Sessions, challenges, and OAuth state are rows with expiry timestamps. The connection pool (`pgxpool`) is configured with explicit bounds (max 10 locally, min 0) and timeouts.

No Redis, Memcached, or any external cache is introduced.

## Consequences

- Rate limit counters under very high concurrency may show slight over-counting due to window boundary races; this is acceptable for the threat model.
- DB query latency directly affects auth endpoint latency; the 3-second per-operation timeout is the guard.
- Future phases may reconsider if write volume justifies a cache layer; that requires a new ADR and owner approval.
