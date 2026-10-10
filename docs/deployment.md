# Deployment

## Render.com (`render.yaml` blueprint)

1. Connect your GitHub repository.
2. Create a new Web Service (or deploy the `render.yaml` blueprint).
3. Set environment variables:
   - `PORT=8081` (or keep default)
   - `DATABASE_URL=postgresql://...` (Neon URL with `?sslmode=require`, `sync: false`)
   - `NEON_AUTH_BASE_URL=https://...` (same Neon branch, `sync: false`)
4. Build command: `go build -o snap-link`
5. Start command: `./snap-link`
6. Run migrations against the branch **before** first boot (see [Setup](setup.md)) — the app fatals if `links` is missing.

Health check path: `/health` (pings Postgres, `503` when unreachable).

## Docker (Local Development)

```bash
docker compose up -d --build
# App: http://localhost:8081/health
```

`docker-compose.yml` expects `DATABASE_URL` + `NEON_AUTH_BASE_URL` in the environment (external Neon DB — no local Postgres service).

## Manual VPS Deployment

```bash
# Build
go build -o snap-link

# Set environment and run
export DATABASE_URL="postgresql://..."
export NEON_AUTH_BASE_URL="https://..."
export PORT=8081
./snap-link
```

Terminate with `SIGINT`/`SIGTERM` — the server drains in-flight requests (10s timeout).

## Environment Variables

| Variable | Default | Required | Description |
|----------|---------|----------|-------------|
| `PORT` | `8081` | No | HTTP server port |
| `DATABASE_URL` | — | Yes | PostgreSQL connection string |
| `NEON_AUTH_BASE_URL` | — | Yes | Neon Managed Better Auth base URL (per-branch); JWKS + issuer derived |

## Production Considerations

- `POST /snap` requires a Neon JWT (`Authorization: Bearer`); per-IP rate limiting (5/min) still applies.
- Enable SSL/TLS via reverse proxy (nginx, Caddy).
- Monitor the database connection pool (capped at 25 open / 5 idle in `main.go`).
- Regular database backups; never run migration `down` on prod blindly.
- Monitor error rates and latencies; `401` spikes mean expired/invalid tokens, `429` spikes mean legitimate demand or abuse.
