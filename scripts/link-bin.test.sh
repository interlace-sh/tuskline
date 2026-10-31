#!/usr/bin/env bash
# SPDX-License-Identifier: AGPL-3.0-only
#
# Self-test for link-bin.sh. Uses a throwaway directory, never ~/.local/bin.
set -euo pipefail

here="$(cd "$(dirname "$0")" && pwd)"
link="$here/link-bin.sh"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
failures=0

fail() {
	echo "FAIL: $*" >&2
	failures=$((failures + 1))
}

# A fake checkout: bin/tl is the only layout the linker accepts.
mkdir -p "$tmp/repo/bin" "$tmp/dest"
PATH="$tmp/dest:$PATH"
printf '#!/bin/sh\necho tl\n' >"$tmp/repo/bin/tl"
chmod +x "$tmp/repo/bin/tl"

out="$(bash "$link" "$tmp/repo/bin/tl" "$tmp/dest")"
want="$tmp/dest/tl -> $tmp/repo/bin/tl"
if [ "$out" != "$want" ]; then
	fail "create: got [$out] want [$want]"
fi
if [ "$(readlink "$tmp/dest/tl")" != "$tmp/repo/bin/tl" ]; then
	fail "create: link target is $(readlink "$tmp/dest/tl")"
fi

# Rebuilding overwrites bin/tl. The same link must still name that file.
printf '#!/bin/sh\necho rebuilt\n' >"$tmp/repo/bin/tl"
chmod +x "$tmp/repo/bin/tl"
bash "$link" "$tmp/repo/bin/tl" "$tmp/dest" >/dev/null
if [ "$(readlink "$tmp/dest/tl")" != "$tmp/repo/bin/tl" ]; then
	fail "rebuild: link no longer tracks bin/tl"
fi

# A real file is not replaced.
rm "$tmp/dest/tl"
printf 'other\n' >"$tmp/dest/tl"
if bash "$link" "$tmp/repo/bin/tl" "$tmp/dest" >/dev/null 2>"$tmp/err"; then
	fail "regular file: linker overwrote it"
fi
if ! grep -q "not a symlink" "$tmp/err"; then
	fail "regular file: missing explanation: $(cat "$tmp/err")"
fi
if [ "$(cat "$tmp/dest/tl")" != "other" ]; then
	fail "regular file: contents changed"
fi

# A link to somewhere else is not replaced.
rm "$tmp/dest/tl"
ln -s /usr/bin/true "$tmp/dest/tl"
if bash "$link" "$tmp/repo/bin/tl" "$tmp/dest" >/dev/null 2>"$tmp/err"; then
	fail "foreign link: linker replaced it"
fi
if [ "$(readlink "$tmp/dest/tl")" != /usr/bin/true ]; then
	fail "foreign link: target changed to $(readlink "$tmp/dest/tl")"
fi

# A link from an older checkout of the same layout is updated.
rm "$tmp/dest/tl"
mkdir -p "$tmp/old/bin"
ln -s "$tmp/old/bin/tl" "$tmp/dest/tl"
bash "$link" "$tmp/repo/bin/tl" "$tmp/dest" >/dev/null
if [ "$(readlink "$tmp/dest/tl")" != "$tmp/repo/bin/tl" ]; then
	fail "moved checkout: link still points at $(readlink "$tmp/dest/tl")"
fi

if [ "$failures" -ne 0 ]; then
	echo "$failures link-bin check(s) failed" >&2
	exit 1
fi
echo "link-bin checks passed"
