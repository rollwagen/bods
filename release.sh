#!/usr/bin/env bash
#
# Usage: VERSION=0.0.1 release.sh
#

set -euf -o pipefail

if ! echo "${VERSION}" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+(-rc\.[0-9]+)?$'; then
	echo "$VERSION is not in MAJOR.MINOR.PATCH format"
	exit 1
fi

# Create a new tag and push it, this will trigger the goreleaser workflow in .github/workflows/goreleaser.yml
git tag "${VERSION}" -a -m "release v${VERSION}"
#
# Push the tag and the commit if the commit was or wasn't pushed before
git push --atomic origin main "${VERSION}"
