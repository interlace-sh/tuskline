#!/usr/bin/env bash
# SPDX-License-Identifier: AGPL-3.0-only
#
# Container image smoke test (roadmap M0.9 acceptance): builds the image,
# runs it on a random local port and checks that
#   - /api/v1/health returns {"status":"ok"},
#   - the embedded web UI is served at /,
#   - the container runs as a non-root user.
# Needs Docker. Usage: scripts/image-smoke.sh [image tag]
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
tag="${1:-tuskline:smoke}"
name="tuskline-smoke-$$"

cleanup() { docker rm -f "$name" >/dev/null 2>&1 || true; }
trap cleanup EXIT

fail() {
	echo "image smoke: $*" >&2
	docker logs "$name" >&2 2>&1 || true
	exit 1
}

docker build --quiet -t "$tag" \
	--build-arg VERSION="$(git -C "$root" describe --tags --always --dirty 2>/dev/null || echo 0.0.0-dev)" \
	--build-arg COMMIT="$(git -C "$root" rev-parse --short HEAD 2>/dev/null || echo none)" \
	--build-arg DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
	"$root" >/dev/null

user="$(docker image inspect --format '{{.Config.User}}' "$tag")"
case "$user" in
"" | root | 0 | 0:0 | root:root) fail "image runs as root (User=\"$user\")" ;;
esac

docker run -d --name "$name" -p 127.0.0.1::8080 "$tag" >/dev/null
port="$(docker port "$name" 8080/tcp | head -n 1 | sed 's/.*://')"
url="http://127.0.0.1:$port"

# Poll with a deadline rather than sleeping a fixed time.
deadline=$((SECONDS + 30))
body=""
until body="$(curl -fsS "$url/api/v1/health" 2>/dev/null)"; do
	[ "$SECONDS" -lt "$deadline" ] || fail "no response from $url/api/v1/health within 30 s"
	sleep 0.2
done

compact="$(printf '%s' "$body" | tr -d ' \n')"
[ "$compact" = '{"status":"ok"}' ] || fail "health returned '$body', want {\"status\":\"ok\"}"

curl -fsS "$url/" | grep -q '<!doctype html>' || fail "web UI not served at /"

echo "image smoke: ok ($tag, user $user, health $compact)"
