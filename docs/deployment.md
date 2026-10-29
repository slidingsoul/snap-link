# Deployment

## Before you deploy

**Apply migrations first.** The app never runs DDL and fatals on boot if the
`links` table is missing, so a deploy against an unmigrated branch fails.

Use the **direct** (non-`-pooler`) connection string for migrations: they take
a session-level advisory lock, which PgBouncer transaction mode can break. Use
the **pooled** string (host contains `-pooler`) for the app itself.

```bash
go build -tags 'postgres' -o /tmp/migrate github.com/golang-migrate/migrate/v4/cmd/migrate
/tmp/migrate -path migrations -database "$DIRECT_DATABASE_URL" up
```

Run this from your laptop or CI; Render's runtime image has no Go toolchain, so
migrations are intentionally not a `preDeployCommand`.

## Render.com (`render.yaml` blueprint)

1. Connect your GitHub repository.
2. Create a new Web Service (or deploy the `render.yaml` blueprint).
3. Set environment variables (all `sync: false`):
   - `DATABASE_URL` — Neon **pooled** URL with `?sslmode=require`
   - `NEON_AUTH_BASE_URL` — same Neon branch, Console → Branch → Auth
   - `CORS_ALLOWED_ORIGINS` — e.g. `https://app.example.com` (required for a browser frontend)
   - `PUBLIC_BASE_URL` — e.g. `https://snap-link.onrender.com` (used in `POST /snap` responses)
4. Build command: `go build -o snap-link`
5. Start command: `./snap-link`

Do **not** set `PORT`: Render injects the port it routes to.

Health check path: `/health` (pings Postgres, `503` when unreachable, so the
instance is pulled from rotation while the DB is down). `/livez` is the
dependency-free liveness probe used by the container runtime.

The app retries the initial Neon connection (5 attempts, linear backoff) so a
scale-to-zero cold start doesn't fail the deploy.

## Google App Engine (standard)

App Engine builds the Go module directly — the `Dockerfile` is ignored — and
injects `PORT`, which `main.go` already honors.

One-time project setup (the region is permanent):

```bash
gcloud config set project PROJECT_ID
gcloud app create --region=asia-southeast1   # co-locate with your Neon branch
gcloud services enable appengine.googleapis.com cloudbuild.googleapis.com
```

Then deploy:

```bash
cp .env.deploy.example .env.deploy   # pooled DATABASE_URL, auth, direct URL for migrations
chmod 600 .env.deploy
./deploy.sh
```

`deploy.sh` renders `app.yaml` from `app.yaml.example` (gitignored, because App
Engine only supports plaintext `env_variables`), applies migrations when
`DIRECT_DATABASE_URL` is set, then runs `gcloud app deploy`.
`./deploy.sh --render-only` writes `app.yaml` without deploying.

Notes:

- `runtime: go127`; the `go 1.25.11` directive in `go.mod` is satisfied.
- Do **not** set `PORT` in `app.yaml` — App Engine reserves it.
- `PUBLIC_BASE_URL` is optional: `deploy.sh` defaults it to
  `https://<defaultHostname>`.
- `CORS_ALLOWED_ORIGINS` is optional, but empty rejects all cross-origin
  browser requests. Set it to the frontend origin.
- `DIRECT_DATABASE_URL` is optional; if empty, migrations are skipped. The app
  fatals on boot if `links` is missing.
- `max_instances: 3` caps database connections (25 per instance) and the
  effective per-user rate limit (5 per instance). Raise deliberately.
- Add the App Engine URL to **Neon Console → Branch → Auth → trusted domains**
  or Better Auth rejects the smoke test's `Origin`.
- Deploys use your local working tree, not the pushed commit. Use a Cloud Build
  trigger with `gcloud app deploy` for push-to-deploy.
- Values in the deployed `app.yaml` are visible to project viewers. If that
  matters, read the two secrets from Secret Manager in code instead.

Verify:

```bash
BASE="https://$(gcloud app describe --format='value(defaultHostname)')"
curl -fsS "$BASE/livez";  echo
curl -fsS "$BASE/health"; echo
gcloud app logs tail -s default
```

## Docker (Local Development)

```bash
docker compose up -d --build
# App: http://localhost:8081/health
```

`docker-compose.yml` expects `DATABASE_URL` + `NEON_AUTH_BASE_URL` in the
environment (external Neon DB — no local Postgres service). `CORS_ALLOWED_ORIGINS`
and `PUBLIC_BASE_URL` are optional pass-throughs.

The image runs as a non-root user and its `HEALTHCHECK` hits `/livez`.

## Manual VPS Deployment

```bash
# Build
go build -o snap-link

# Set environment and run
export DATABASE_URL="postgresql://..."        # pooled Neon URL
export NEON_AUTH_BASE_URL="https://..."
export CORS_ALLOWED_ORIGINS="https://app.example.com"
export PUBLIC_BASE_URL="https://snap.example.com"
export PORT=8081
./snap-link
```

Terminate with `SIGINT`/`SIGTERM` — the server drains in-flight requests (10s
timeout) and each request has a 5s context deadline for handlers/DB queries.

## Environment Variables

| Variable | Default | Required | Description |
|----------|---------|----------|-------------|
| `PORT` | `8081` | No | HTTP server port (Render injects its own) |
| `DATABASE_URL` | — | Yes | PostgreSQL connection string (pooled URL for the app) |
| `NEON_AUTH_BASE_URL` | — | Yes | Neon Managed Better Auth base URL (per-branch); JWKS + issuer derived |
| `CORS_ALLOWED_ORIGINS` | — | No | Comma-separated browser origins allowed to call `POST /snap`; empty rejects all cross-origin requests |
| `PUBLIC_BASE_URL` | — | No | Public origin used to build shareable links; empty derives from the request Host/Proto |

## Production Considerations

- **CORS is mandatory for a browser frontend.** `POST /snap` sends
  `Authorization`, so the browser preflights with `OPTIONS`. If
  `CORS_ALLOWED_ORIGINS` is unset, every preflight is rejected.
- **Rate limiting is per user, in memory.** Counters are keyed on the JWT `sub`
  (IP fallback) and live in the process: multiple replicas multiply the
  effective limit (`5 × instances`) and a restart resets every window. Move the
  limiter to Redis/Postgres before scaling out.
- **Neon pooler + prepared statements.** If the app logs
  `prepared statement ... already exists`, append
  `?default_query_exec_mode=simple_protocol` to `DATABASE_URL`.
- **Connection pool is per instance**: 25 open / 5 idle. Size it against your
  Neon plan when running multiple replicas.
- Enable SSL/TLS via reverse proxy (nginx, Caddy) or the platform.
- Regular database backups; never run migration `down` on prod blindly.
- Monitor error rates and latencies; `401` spikes mean expired/invalid tokens,
  `429` spikes mean legitimate demand or abuse.
