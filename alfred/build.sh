#!/bin/sh
# Régénère bagent.alfredworkflow (zip de alfred/info.plist) à la racine du repo.
# Le workflow ne contient que le câblage Alfred ; la liste des workspaces est
# lue en direct par `bagent alfred`. À relancer seulement si info.plist change.
set -e

DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(dirname "$DIR")"
OUT="$ROOT/bagent.alfredworkflow"

plutil -lint "$DIR/info.plist" >/dev/null
rm -f "$OUT"
(cd "$DIR" && zip -q -X "$OUT" info.plist)
echo "  ✓ $OUT"
