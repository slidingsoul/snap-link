# Setup

Local setup from zero to running server.

## Prerequisites

- Go 1.21 or higher
- PostgreSQL 12 or higher (or Neon DB)
- `curl` + `jq` for the smoke test

## 1. Clone and Initialize

```bash
git clone <repository-url>
cd snap-link
cp .env.example .env
```

## 2. Configure Environment Variables

Edit `.env`:

```env
PORT=8081
DATABASE_URL=postgres://username:password@localhost:5432/link_shortener
NEON_AUTH_BASE_URL=https://ep-xxxxx.aws.neon.tech/dbname/auth
CORS_ALLOWED_ORIGINS=http://localhost:3000   # browser frontend origin(s), comma-separated
PUBLIC_BASE_URL=http://localhost:8081        # used in POST /snap responses
```

For **Neon DB**:

```env
DATABASE_URL=postgresql://user:password@ep-xxxxx.neon.tech/dbname?sslmode=require
NEON_AUTH_BASE_URL=https://ep-xxxxx.aws.neon.tech/dbname/auth
```

Get the auth URL from Neon Console → Branch → Auth (same branch as `DATABASE_URL`).
JWKS + issuer are derived from it; no secret needed on the server.

`CORS_ALLOWED_ORIGINS` is required for a browser client — `POST /snap` sends
`Authorization`, so the browser preflights with `OPTIONS`. Empty means no
cross-origin requests are allowed. For production use the **pooled** Neon URL
(host contains `-pooler`) for the app and the **direct** URL for migrations.

## 3. Install Dependencies

```bash
go mod download
go mod tidy
```

## 4. Run Migrations

Schema is versioned in `migrations/` (golang-migrate, manual — the app never migrates itself; Neon auth tables are untouched).
Build the migrate CLI once (the `postgres` driver needs its build tag):

```bash
go build -tags 'postgres' -o /tmp/migrate github.com/golang-migrate/migrate/v4/cmd/migrate
/tmp/migrate -path migrations -database "$DATABASE_URL" up       # apply pending migrations
/tmp/migrate -path migrations -database "$DATABASE_URL" version  # show current version
```

Against Neon, point `-database` at the **direct** (non-`-pooler`) URL: the
migration runner holds a session-level advisory lock, which the pooler's
transaction mode can drop.

New migration: create `migrations/<timestamp>_<name>.up.sql` + `.down.sql` (timestamp format `YYYYMMDDHHMMSS`, e.g. `20261010120000_add_owner.up.sql`).

Run `up` once per database/branch before starting the app. The baseline is a safe no-op on databases GORM AutoMigrate already created.

## 5. Run the Application

```bash
go run main.go
```

The server starts on `http://localhost:8081`. It fatals on boot if `DATABASE_URL`, `NEON_AUTH_BASE_URL`, or the `links` table is missing.

## 6. Smoke Test

End-to-end script (`curl` + `jq`, aborts on first failure):

```bash
export NEON_AUTH_BASE_URL="https://ep-xxx.../neondb/auth"
./snap-link-test.sh
```

Covers register → login → JWT → `POST /snap` (`201`) → `302` redirect → negatives (`401`/`400`/`404`) → rate-limit `429`. See also [Development](development.md).
