#!/usr/bin/env bash
# SPDX-License-Identifier: AGPL-3.0-only
#
# Point a directory on PATH at a just-built tl or tuskline binary.
# The link stays valid across rebuilds because it names the file in bin/,
# which the build overwrites. A regular file, or a link to somewhere else,
# is left alone.
# Usage: scripts/link-bin.sh <binary> <dest-dir>
set -euo pipefail

if [ "$#" -ne 2 ]; then
	echo "usage: link-bin.sh <binary> <dest-dir>" >&2
	exit 2
fi

src="$(realpath "$1")"
dest_dir="$2"
name="$(basename "$src")"
dest="$dest_dir/$name"

case "$src" in
*/bin/"$name") ;;
*)
	echo "link-bin: $src is not bin/$name" >&2
	exit 1
	;;
esac

if [ ! -x "$src" ] || [ -d "$src" ]; then
	echo "link-bin: $src is not an executable" >&2
	exit 1
fi

mkdir -p "$dest_dir"

if [ -L "$dest" ]; then
	current="$(readlink "$dest")"
	case "$current" in
	*/bin/"$name") ;;
	*)
		echo "link-bin: $dest already points at $current" >&2
		exit 1
		;;
	esac
elif [ -e "$dest" ]; then
	echo "link-bin: $dest already exists and is not a symlink" >&2
	exit 1
fi

ln -sfn "$src" "$dest"

case ":${PATH:-}:" in
*":$dest_dir:"*) ;;
*)
	echo "link-bin: $dest_dir is not on PATH" >&2
	;;
esac

echo "$dest -> $src"
