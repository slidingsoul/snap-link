# Architecture

## Project Structure

```
.
├── main.go                      # Application entry point, middleware chain, graceful shutdown
├── go.mod                       # Go module dependencies
├── .env.example                 # Environment variables template
├── migrations/                  # Versioned schema (golang-migrate, manual)
├── middleware/
│   ├── rate_limit.go            # Fixed per-user limiter (internal/ratelimit)
│   ├── timeout.go               # 5s request context deadline
│   └── require_auth.go          # Neon JWT gate for POST /snap
├── internal/
│   ├── auth/                    # Neon JWT verifier (issuer-only, JWKS derived)
│   ├── shortcode/               # Single source: Charset, Generate, Validate
│   └── ratelimit/               # Fixed limit: DefaultPerMinute = 5
├── models/
│   └── link.go                  # Link model and request/response DTOs
├── repository/
│   └── link_repository.go       # Database operations layer
├── service/
│   └── link_service.go          # Business logic + URL validation (delegates code check)
└── controller/
    └── link_controller.go       # HTTP handlers layer
```

Layering: Controller → Service → Repository. Controllers map errors to HTTP statuses without leaking internals (`400` validation, `404` not-found, `500` infra).

## Database Schema

Owned by `migrations/` — the app never runs DDL (see [Setup](setup.md)):

```sql
CREATE TABLE links (
  id bigserial PRIMARY KEY,
  original_url text NOT NULL,
  short_code varchar(10) NOT NULL UNIQUE,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX uni_links_short_code ON links (short_code);
CREATE INDEX idx_links_short_code ON links (short_code);
CREATE INDEX idx_links_original_url ON links (original_url);
```

A legacy `lookups` table may exist in older databases; it is ignored. Neon auth tables (`neon_auth.*`) are managed by Neon.

## Short Code Generation

- **Length**: 6 characters (`internal/shortcode.Length`)
- **Alphabet**: A-Z, a-z, 0-9 (62 characters, `internal/shortcode.Charset`)
- **Randomness**: `crypto/rand` with rejection sampling (no modulo bias)
- **Collision handling**: unique constraint + up to 5 automatic retries
- **Combinations**: ~62^6 ≈ 56 trillion
- **Validation**: `shortcode.Validate()` (4–10 chars from `Charset`) is the single source — repository generation and service validation both use it, so generated codes always validate.

## Authentication & Rate Limiting

- `POST /snap` chain: `RequireAuth` (Neon JWT) → fixed-window limiter → handler. The limiter runs **after** auth and keys on the JWT `sub` (IP fallback), so rejected `401`s don't consume quota and rotating `X-Forwarded-For` doesn't reset it. `429` carries `Retry-After`.
- Limiter state is in-process: the effective limit is `5 × instances`, and a restart clears every window. Move it to Redis/Postgres before scaling out.
- JWT verification: EdDSA signature via `<NEON_AUTH_BASE_URL>/.well-known/jwks.json`, issuer = origin of base URL, plus expiry. No audience check. `sub` claim becomes `c.Locals("userID")` (not persisted on `links` yet).
- Public reads: `GET /:code`, `GET /health` (DB readiness), `GET /livez` (process liveness).
- Browser access: CORS is configured from `CORS_ALLOWED_ORIGINS`; empty means no cross-origin. `POST /snap` needs an `OPTIONS` preflight because of the `Authorization` header.
- Request lifetime: `middleware.Timeout` attaches a 5s deadline to the context; every repository call uses `db.WithContext(ctx)`, so a slow query is cancelled instead of pinning a connection.
- Server hardening: `helmet`, `requestid` (echoed as `X-Request-ID`), `BodyLimit 16KB`, read/write/idle timeouts, no `ServerHeader` leak, `ProxyHeader: X-Forwarded-For` so IPs are real behind Render.
- `POST /snap` returns the full URL built from `PUBLIC_BASE_URL` (fallback: request Host/Proto) plus `short_code`.

## Logging

Request/response logging with status, latency, and request ID (`main.go` logger format). Errors log server-side with detail; clients get generic messages (`Invalid URL format`, `Failed to create short link`) — no DB internals leak.

## Performance Notes

- Short code generation: < 1ms
- Database insert: ~5–10ms (without network latency)
- Average response time: 10–20ms
- Connection pool: 25 max open / 5 idle / 5-min lifetime, **per instance**
- Initial connection: retried up to 5× with linear backoff (Neon cold start)
