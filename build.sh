#!/bin/sh
# Usage: ./build.sh [US|FR|DE] [amd64|arm64]
set -e
SITE=${1:-US}
ARCH=${2:-amd64}
TAG=werp-$(echo "$SITE" | tr 'A-Z' 'a-z'):latest
docker build . -t "$TAG" --platform "linux/$ARCH" --build-arg SITE="$SITE"
echo "Run: docker run -ti --rm -p 8080:8080 $TAG"
