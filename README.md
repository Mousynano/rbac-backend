# RBAC Blueprint Designer API

Go API for platform accounts and owner-scoped RBAC blueprint design. The repository is in Phase 1 development: health endpoints and database schema are available; authentication and diagram routes are not yet implemented. See [ARCHITECTURE.md](ARCHITECTURE.md) and the specification under `.kiro/specs/phase-1-bootstrap/`.

## Local bootstrap

Requires Go 1.27.4, Docker Engine with Compose, and a local shell. Copy `.env.example` to `.env`. Replace `POSTGRES_PASSWORD` with a random local password and use the same value inside `DATABASE_URL`. Generate `OTP_HMAC_KEY` as base64 of at least 32 cryptographically random bytes. For example, in PowerShell:

```powershell
Copy-Item .env.example .env
$rng = [Security.Cryptography.RandomNumberGenerator]::Create()
$bytes = New-Object byte[] 32
$rng.GetBytes($bytes)
$rng.Dispose()
[Convert]::ToBase64String($bytes)
```

Paste the generated key into `.env`, then run:

```powershell
docker compose up --build -d
docker compose ps
Invoke-RestMethod http://localhost:8080/healthz
Invoke-RestMethod http://localhost:8080/readyz
```

Mailpit's local inbox is at `http://localhost:8025`. The API is bound to loopback port 8080. PostgreSQL is internal to Compose. The `migrate` one-shot service must complete successfully before the API starts. Inspect logs with `docker compose logs migrate api` if readiness fails. `go test ./...` runs local unit tests without Docker.

## Migration and operations

Migrations are versioned SQL in `migrations/`. The API never auto-migrates. Run `docker compose run --rm migrate status` or `docker compose run --rm migrate up` after changing migrations. The migration CLI permits only `up`, `status`, and `version`; destructive down migrations require an explicit reviewed recovery plan. Back up PostgreSQL before deploying schema changes. The `admin` command remains a stub until the email policy milestone.

## Development gates

`make verify` lists the intended gates. Integration suites, CI, and `govulncheck` setup are pending; a green local unit test is not Phase 1 acceptance. Manual Google and production SMTP smoke tests require configured external infrastructure and are tracked separately.
