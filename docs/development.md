# Development

## Unit & Integration Tests

```bash
go test ./...        # all packages (SQLite in-memory, no external DB needed)
go vet ./...         # static checks
go fmt ./...         # formatting
```

Controller tests call handlers directly (bypassing `RequireAuth` middleware — auth is covered by `middleware/require_auth_test.go` plus `internal/auth` roundtrip tests against a stub JWKS).

## End-to-End Smoke Test

`./snap-link-test.sh` (`curl` + `jq`, `set -e`, aborts on first failure). Requires the server running with real Neon credentials:

```bash
export NEON_AUTH_BASE_URL="https://ep-xxx.../neondb/auth"
./snap-link-test.sh
# overrides: API_BASE=... AUTH_ORIGIN=... TEST_EMAIL=... TEST_PASSWORD=... EXPECTED_LIMIT=5
```

Flow: health → register → login → JWT → create (`201`) → redirect (`302` + `Location`) → negatives (`401`/`400`/`404`) → rate-limit probes (`EXPECTED_LIMIT` aware — restart the server first for a clean fixed window).

The limiter runs after auth and is keyed on the JWT `sub`, so the script assumes
two quota-consuming requests before the probes (create + bad-URL `400`); the
no-token `401` does not count. Fewer than `EXPECTED_LIMIT - 2` probes means the
window was already used — restart the server.

## Adding Migrations

```bash
# create migrations/<timestamp>_<name>.up.sql + .down.sql, then:
/tmp/migrate -path migrations -database "$DATABASE_URL" up
```

Never edit an applied migration; add a new one. See [Setup](setup.md).

## Style

- Keep `README.md` slim — details belong in `docs/`.
- API behavior changes update `docs/api.md` in the same commit.
- Commit messages follow conventional prefixes (`feat:`, `fix:`, `refactor:`, `docs:`, `test:`).

## Contributing

1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

## License

MIT
