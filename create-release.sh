#!/usr/bin/env bash

VERSION=$1

if [ -z "$VERSION" ]; then
  echo "Usage: create-release.sh VERSION"
  exit 1
fi

echo "Creating release for version $VERSION"

git tag -a $VERSION -m "new release"
git push origin $VERSION

goreleaser release --rm-dist