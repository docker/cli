#!/usr/bin/env bash
#
# This script is used to coerce certain commands which rely on the presence of
# a go.mod into working with our repository. It works by creating a fake
# go.mod, running a specified command (passed via arguments), and removing it
# when the command is finished. This script should be dropped when this
# repository is a proper Go module with a permanent go.mod.

set -euo pipefail

SCRIPTDIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOTDIR="$(cd "${SCRIPTDIR}/.." && pwd)"

cleanup_paths=()

create_symlink() {
	local target="$1"
	local link="$2"

	if [ -e "$link" ]; then
		# see https://superuser.com/a/196698
		if ! [ "$link" -ef "${ROOTDIR}/${target}" ]; then
			echo "$(basename "$0"): WARN: $link exists but is not the expected symlink!" >&2
			echo "$(basename "$0"): WARN: Using your version instead of our generated version -- this may misbehave!" >&2
		fi
		return
	fi

	set -x
	ln -s "$target" "$link"
	cleanup_paths+=( "$link" )
}

create_symlink "vendor.mod" "${ROOTDIR}/go.mod"
create_symlink "vendor.sum" "${ROOTDIR}/go.sum"

if [ "${#cleanup_paths[@]}" -gt 0 ]; then
	trap 'rm -f "${cleanup_paths[@]}"' EXIT
fi

GO111MODULE=on GOTOOLCHAIN=local "$@"
