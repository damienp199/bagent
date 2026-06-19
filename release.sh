#!/bin/sh
# Publie une release bagent : compile les deux cibles macOS avec la version
# embarquée (ldflags), puis crée la release GitHub (le tag devient "latest").
# Le ldflags est indispensable : sans lui, bagent --version affiche "dev" et la
# notif d'auto-update ne se déclenche jamais chez les clients.
# Usage : ./release.sh vX.Y.Z "notes de release"
set -e

TAG="$1"
NOTES="${2:-}"
[ -n "$TAG" ] || { echo "usage: ./release.sh vX.Y.Z [notes]" >&2; exit 1; }

LD="-X github.com/damienp199/bagent/internal/app.version=$TAG"
mkdir -p dist
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags "$LD" -o dist/bagent-darwin-arm64 .
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "$LD" -o dist/bagent-darwin-amd64 .
gh release create "$TAG" dist/bagent-darwin-arm64 dist/bagent-darwin-amd64 \
  --target main --title "$TAG" --notes "$NOTES"
