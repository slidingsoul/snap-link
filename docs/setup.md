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
```

For **Neon DB**:

```env
DATABASE_URL=postgresql://user:password@ep-xxxxx.neon.tech/dbname?sslmode=require
NEON_AUTH_BASE_URL=https://ep-xxxxx.aws.neon.tech/dbname/auth
```

Get the auth URL from Neon Console → Branch → Auth (same branch as `DATABASE_URL`).
JWKS + issuer are derived from it; no secret needed on the server.

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
