# Snap Link

A URL shortening service built with Go, Fiber, and PostgreSQL. `POST /snap` requires a Neon JWT; reads are public and `GET /:code` issues a `302` redirect.

## Features

- Collision-resistant 6-character short codes (`crypto/rand`, retry on conflict)
- Strict URL validation (http/https only)
- Neon Managed Better Auth (EdDSA JWT, issuer-only verification)
- Fixed per-IP rate limiting (5/min)
- Versioned Postgres migrations (manual, app never migrates itself)

## Quickstart

```bash
cp .env.example .env   # set DATABASE_URL + NEON_AUTH_BASE_URL (same Neon branch)
go mod download
go build -tags 'postgres' -o /tmp/migrate github.com/golang-migrate/migrate/v4/cmd/migrate
/tmp/migrate -path migrations -database "$DATABASE_URL" up
go run main.go         # http://localhost:8081
```

Export `NEON_AUTH_BASE_URL` (Console → Branch → Auth), then `./snap-link-test.sh` for the end-to-end smoke test.

## Docs

- [Setup](docs/setup.md) — local setup, migrations, smoke test
- [API](docs/api.md) — endpoints, auth, rate limits, errors
- [Deployment](docs/deployment.md) — Render, Docker, VPS, environment variables
- [Architecture](docs/architecture.md) — structure, schema, short codes, auth design
- [Development](docs/development.md) — tests, migrations workflow, contributing
