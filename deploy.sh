#!/usr/bin/env bash
# Deploy snap-link to Google App Engine (standard environment).
#
#   cp .env.deploy.example .env.deploy   # fill in DB/auth/CORS/base URL
#   ./deploy.sh                          # render app.yaml, migrate, deploy
#   ./deploy.sh --render-only            # write app.yaml and stop
#
# app.yaml is generated from app.yaml.example and is gitignored: App Engine
# only supports plaintext env_variables, so the rendered file holds secrets.
set -euo pipefail

cd "$(dirname "$0")"

ENV_FILE="${ENV_FILE:-.env.deploy}"
TEMPLATE="app.yaml.example"
RENDERED="app.yaml"
RENDER_ONLY=0
[ "${1:-}" = "--render-only" ] && RENDER_ONLY=1

gcloud_cmd() {
	if [ -n "${PROJECT_ID:-}" ]; then
		gcloud --project="$PROJECT_ID" "$@"
	else
		gcloud "$@"
	fi
}

is_set() {
	local name="$1"
	eval "test -n \"\${$name:-}\""
}

if [ ! -f "$ENV_FILE" ]; then
	echo "Missing $ENV_FILE. Run: cp .env.deploy.example $ENV_FILE" >&2
	exit 1
fi

set -a
# shellcheck disable=SC1090
. "$ENV_FILE"
set +a

missing=0
for var in DATABASE_URL NEON_AUTH_BASE_URL; do
	if ! is_set "$var"; then
		echo "Missing $var in $ENV_FILE" >&2
		missing=1
	fi
done
[ "$missing" -eq 0 ] || exit 1

if ! is_set CORS_ALLOWED_ORIGINS; then
	echo "! CORS_ALLOWED_ORIGINS is empty — cross-origin browser calls will be rejected."
fi
if ! is_set DIRECT_DATABASE_URL; then
	echo "! DIRECT_DATABASE_URL is empty — migrations will be skipped."
fi

if ! command -v envsubst >/dev/null 2>&1; then
	echo "envsubst not found (brew install gettext)" >&2
	exit 1
fi

if [ "$RENDER_ONLY" -eq 0 ]; then
	if ! command -v gcloud >/dev/null 2>&1; then
		echo "gcloud not found" >&2
		exit 1
	fi

	if ! gcloud_cmd app describe >/dev/null 2>&1; then
		echo "No App Engine app in project '${PROJECT_ID:-$(gcloud config get-value project 2>/dev/null)}'." >&2
		echo "Create one first (the region is permanent):" >&2
		echo "  gcloud app create --region=asia-southeast1" >&2
		exit 1
	fi

	# PUBLIC_BASE_URL is optional: default it to the app's hostname so links
	# point at App Engine's public origin.
	if ! is_set PUBLIC_BASE_URL; then
		PUBLIC_BASE_URL="https://$(gcloud_cmd app describe --format='value(defaultHostname)')"
		export PUBLIC_BASE_URL
		echo "→ PUBLIC_BASE_URL defaulted to $PUBLIC_BASE_URL"
	fi
fi

echo "→ Rendering $RENDERED from $TEMPLATE"
umask 077
envsubst '${DATABASE_URL} ${NEON_AUTH_BASE_URL} ${CORS_ALLOWED_ORIGINS} ${PUBLIC_BASE_URL}' \
	<"$TEMPLATE" >"$RENDERED"

if [ "$RENDER_ONLY" -eq 1 ]; then
	echo "Rendered $RENDERED (not deploying)."
	exit 0
fi

# Migrations run before the new code goes live, against the direct URL
# (the pooler drops the session-level advisory lock golang-migrate uses).
if is_set DIRECT_DATABASE_URL; then
	echo "→ Applying migrations (direct URL)"
	mkdir -p bin
	if [ ! -x bin/migrate ]; then
		go build -tags 'postgres' -o bin/migrate github.com/golang-migrate/migrate/v4/cmd/migrate
	fi
	bin/migrate -path migrations -database "$DIRECT_DATABASE_URL" up
else
	echo "! DIRECT_DATABASE_URL not set — skipping migrations."
	echo "  The app fatals on boot if the links table is missing."
fi

echo "→ Deploying to App Engine"
gcloud_cmd app deploy "$RENDERED" --quiet

HOST="$(gcloud_cmd app describe --format='value(defaultHostname)')"
echo "✅ Deployed: https://$HOST"
echo "   curl -fsS https://$HOST/health"
