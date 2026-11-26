# Postman local API collection

Import `RBAC-Backend.postman_collection.json` and `Local.postman_environment.json` into Postman. Select the `RBAC Backend - Local` environment. Set `base_url` to `http://localhost:8080` and `allowed_origin` to the same origin. Start the stack with `docker compose up --build -d`.

The collection includes implemented requests for `GET /healthz`, `GET /readyz`, email registration, email/password login, OTP verification, and guarded `GET /api/v1/auth/me`. OTP requires Mailpit access and a pending cookie established by register/login. Google SSO, OTP resend, password reset, logout, and diagram routes remain planned; they are not implemented and must not be counted as passing tests.

For mutating requests, the collection sends the configured `Origin` and `X-CSRF-Protection: 1`. Postman manages cookies in its cookie jar for the selected host. Once pending/session cookies are implemented, registration/login/OTP requests should share that jar. Do not paste real passwords, OTPs, or production tokens into a shared environment. The example registration uses a reserved `.test` address and is illustrative only; it will fail domain eligibility checks.

The example blueprint follows the v1 shape in the design spec. Replace `diagram_id` with a real response ID and `diagram_version` with its current row version when diagram CRUD is implemented.
