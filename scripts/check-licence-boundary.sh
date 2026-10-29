#!/usr/bin/env bash
# SPDX-License-Identifier: AGPL-3.0-only
#
# Licence boundary check (LICENSING.md, docs/adr/0010-licence-boundary.md).
# Fails when:
#   - a .go file does not start with the SPDX header for its directory
#     (Apache-2.0 in core/, cli/, agent/; AGPL-3.0-only in server/, desktop/);
#   - an Apache module (core, cli, agent) depends on server or desktop, or
#     core depends on cli or agent. Dependencies come from `go list -deps`
#     with tests and the integration tag, so indirect imports count too.
#
# Usage: scripts/check-licence-boundary.sh [repository root]
set -euo pipefail

root="$(cd "${1:-$(dirname "$0")/..}" && pwd)"
mod="github.com/interlace-sh/tuskline"
fail=0

err() {
	echo "licence boundary: $*" >&2
	fail=1
}

spdx_for() {
	case "$1" in
	core | cli | agent) echo "// SPDX-License-Identifier: Apache-2.0" ;;
	*) echo "// SPDX-License-Identifier: AGPL-3.0-only" ;;
	esac
}

# Directories that may not appear in a module's dependency graph.
forbidden_for() {
	case "$1" in
	core) echo "cli agent server desktop" ;;
	cli | agent) echo "server desktop" ;;
	*) echo "" ;;
	esac
}

for dir in core cli agent server desktop; do
	[ -d "$root/$dir" ] || continue
	want="$(spdx_for "$dir")"
	while IFS= read -r -d '' f; do
		first="$(head -n 1 "$f")"
		if [ "$first" != "$want" ]; then
			err "${f#"$root"/}: first line must be '$want'"
		fi
	done < <(find "$root/$dir" -name '*.go' -not -path '*/node_modules/*' -print0)
done

for dir in core cli agent; do
	[ -f "$root/$dir/go.mod" ] || continue
	forbidden="$(forbidden_for "$dir")"
	# -e: report the graph even when a package fails to load (for example an
	# import cycle created by the forbidden import itself); build tasks report
	# load errors separately.
	deps="$(cd "$root/$dir" && go list -e -deps -test -tags integration -f '{{.ImportPath}}' ./...)"
	for bad in $forbidden; do
		hits="$(printf '%s\n' "$deps" | grep -E "^$mod/$bad(/|$)" || true)"
		if [ -n "$hits" ]; then
			err "$dir must not depend on $bad:"
			printf '  %s\n' $hits >&2
		fi
	done
done

if [ "$fail" -ne 0 ]; then
	echo "licence boundary: FAILED (see LICENSING.md)" >&2
	exit 1
fi
echo "licence boundary: ok"
