#!/usr/bin/env bash
# Reports AI-generated code blocks which have not been explicitly approved.
#
# A block starts with a single-line marker such as:
# // AI-GENERATED-BEGIN: v=1; task="add pipeline validation"; reviewer="Phil Mansfield"; tool="Codex"; review="approved"
# and ends with:
# // AI-GENERATED-END

set -euo pipefail

script_dir=$(CDPATH= cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
repo_root=$(CDPATH= cd -- "$script_dir/../.." && pwd)
marker_prefix='AI-GENERATED-BEGIN'
found_unapproved=0

while IFS= read -r marker; do
	if [[ "$marker" != *'review="approved"'* ]]; then
		echo "$marker"
		found_unapproved=1
	fi
done < <(
	find "$repo_root" \
		-type d \( -name .git -o -name vendor \) -prune -o \
		-type f ! -name AGENTS.md \
		-exec grep -nF "${marker_prefix}:" {} + 2>/dev/null || true
)

if ((found_unapproved)); then
	echo 'AI-generated code blocks above are missing review="approved".' >&2
	exit 1
fi
