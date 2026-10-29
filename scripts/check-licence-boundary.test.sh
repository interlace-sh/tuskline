#!/usr/bin/env bash
# SPDX-License-Identifier: AGPL-3.0-only
#
# Self-test for check-licence-boundary.sh: builds a throwaway Go workspace
# and asserts the check passes when clean and fails on a bad import, an
# indirect bad import and a missing or wrong SPDX header.
set -euo pipefail

here="$(cd "$(dirname "$0")" && pwd)"
check="$here/check-licence-boundary.sh"
mod="github.com/interlace-sh/tuskline"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
failures=0

apache="// SPDX-License-Identifier: Apache-2.0"
agpl="// SPDX-License-Identifier: AGPL-3.0-only"

# gofile PATH HEADER [IMPORT]: writes a one-file package, optionally importing IMPORT.
gofile() {
	local path="$1" header="$2" import="${3:-}"
	mkdir -p "$(dirname "$path")"
	{
		echo "$header"
		echo
		echo "package $(basename "$(dirname "$path")")"
		if [ -n "$import" ]; then
			echo
			echo "import _ \"$import\""
		fi
	} >"$path"
}

# fixture: a clean workspace where agent, cli and server import core.
# Workspace modules resolve each other through go.work alone.
fixture() {
	rm -rf "$tmp/ws"
	local d
	for d in core cli agent server; do
		mkdir -p "$tmp/ws/$d"
		printf 'module %s/%s\n\ngo 1.25.0\n' "$mod" "$d" >"$tmp/ws/$d/go.mod"
	done
	printf 'go 1.25.0\n\nuse (\n\t./agent\n\t./cli\n\t./core\n\t./server\n)\n' >"$tmp/ws/go.work"
	gofile "$tmp/ws/core/lib/lib.go" "$apache"
	gofile "$tmp/ws/agent/run/run.go" "$apache" "$mod/core/lib"
	gofile "$tmp/ws/cli/cmd/cmd.go" "$apache" "$mod/core/lib"
	gofile "$tmp/ws/server/api/api.go" "$agpl" "$mod/core/lib"
}

# expect NAME WANT_EXIT PATTERN: runs the check on the fixture.
expect() {
	local name="$1" want="$2" pattern="$3"
	local out code=0
	out="$(GOWORK="$tmp/ws/go.work" GOFLAGS= "$check" "$tmp/ws" 2>&1)" || code=$?
	if [ "$code" -ne "$want" ] || ! printf '%s' "$out" | grep -q -- "$pattern"; then
		echo "FAIL $name: exit $code (want $want), expected output containing '$pattern':" >&2
		printf '%s\n' "$out" | sed 's/^/    /' >&2
		failures=$((failures + 1))
		return
	fi
	echo "ok   $name"
}

fixture
expect "clean workspace passes" 0 "licence boundary: ok"

fixture
gofile "$tmp/ws/cli/cmd/bad.go" "$apache" "$mod/server/api"
expect "cli importing server fails" 1 "cli must not depend on server"

fixture
gofile "$tmp/ws/server/leaf/leaf.go" "$agpl"
gofile "$tmp/ws/core/lib/bad_test.go" "$apache" "$mod/server/leaf"
expect "core test importing server fails" 1 "core must not depend on server"

fixture
gofile "$tmp/ws/cli/cmd/bad.go" "$apache" "$mod/server/api"
gofile "$tmp/ws/agent/run/indirect.go" "$apache" "$mod/cli/cmd"
expect "agent depending on server via cli fails" 1 "agent must not depend on server"

fixture
gofile "$tmp/ws/agent/leaf/leaf.go" "$apache"
gofile "$tmp/ws/core/lib/up.go" "$apache" "$mod/agent/leaf"
expect "core importing agent fails" 1 "core must not depend on agent"

fixture
printf 'package lib\n' >"$tmp/ws/core/lib/noheader.go"
expect "missing SPDX header fails" 1 "core/lib/noheader.go"

fixture
gofile "$tmp/ws/server/api/wrong.go" "$apache"
expect "Apache header in server fails" 1 "server/api/wrong.go"

if [ "$failures" -ne 0 ]; then
	echo "$failures licence boundary self-test(s) failed" >&2
	exit 1
fi
echo "licence boundary self-test: ok"
