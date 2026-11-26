# ADR-0001 — Go toolchain version

**Date:** 2026-11-24  
**Status:** Accepted  
**Deciders:** Product owner, architect  

## Context

The initial `go.mod` declared `go 1.26.4`, matching the developer's locally installed toolchain. The engineering steering (`tech.md`) mandates the **Go 1.27 release line**. Go 1.27.4 was released on 2026-11-10 and contains 12 security fixes, including:

- CVE-2026-97033: build-time arbitrary code execution via malicious cgo directives.
- Compiler and linker flag-injection fixes.
- A `net/http` server crash fix.

With `GOTOOLCHAIN=auto` (the project default), the `go` directive in `go.mod` controls which toolchain actually builds the binary; a module declaring `go 1.26.4` while `GOTOOLCHAIN=auto` is set would silently build with 1.26 and miss these fixes.

## Decision

Set the `go` directive in `go.mod` to `go 1.27.4`. Pin the same version in `Dockerfile` (`golang:1.27.4-bookworm` build stage) and document it here.

## Consequences

- The developer's local `go1.26.4` install will trigger an automatic download of `go1.27.4` on the first `go build` or `go test`, which requires internet access.
- All CI and Docker build environments must use `go1.27.4` or later within the 1.27 line.
- When a newer patch (1.27.5+) is released, update `go.mod`, `Dockerfile`, and this record.
- Go 1.27 continues to honor the Go 1 compatibility promise; no application code changes are required.
