#!/bin/sh
# Régénère bagent.alfredworkflow (zip de alfred/info.plist) à la racine du repo.
# Le workflow ne contient que le câblage Alfred ; la liste des workspaces est
# lue en direct par `bagent alfred`. À relancer seulement si info.plist change.
#
# Usage : build.sh [--reimport]
#   --reimport  ouvre le .alfredworkflow après le build pour forcer Alfred à
#               recharger sa définition (Alfred ne relit pas un info.plist
#               écrasé sous ses pieds : sans réimport, l'ancienne version reste
#               active en mémoire). Nécessite une action manuelle « Import ».
set -e

DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(dirname "$DIR")"
OUT="$ROOT/bagent.alfredworkflow"

plutil -lint "$DIR/info.plist" >/dev/null
rm -f "$OUT"
(cd "$DIR" && zip -q -X "$OUT" info.plist)
echo "  ✓ $OUT"

if [ "$1" = "--reimport" ]; then
	open "$OUT"
	echo "  → Fenêtre Alfred ouverte : clique « Import » pour recharger le workflow."
fi
