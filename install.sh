#!/bin/sh
# Installe bagent. Usage :
#   curl -fsSL https://raw.githubusercontent.com/damienp199/bagent/main/install.sh | sh
# ou, depuis un clone du repo (compile depuis les sources, nécessite Go) :
#   ./install.sh
set -e

REPO="damienp199/bagent"
BIN="bagent"
DEST="${BAGENT_BIN:-$HOME/.local/bin}"

err() { printf '  \033[31m✗\033[0m %s\n' "$1" >&2; exit 1; }
ok()  { printf '  \033[32m✓\033[0m %s\n' "$1"; }

[ "$(uname -s)" = "Darwin" ] || err "bagent ne supporte que macOS"
mkdir -p "$DEST"
STAGE="$DEST/.$BIN.new"

# Mode dev : si on est dans le repo et que Go est présent, compiler depuis les sources.
if [ -f "go.mod" ] && grep -q "damienp199/bagent" go.mod 2>/dev/null && command -v go >/dev/null 2>&1; then
  printf '  Compilation depuis les sources…\n'
  go build -o "$STAGE" . || err "échec de la compilation"
else
  case "$(uname -m)" in
    arm64)  asset="bagent-darwin-arm64" ;;
    x86_64) asset="bagent-darwin-amd64" ;;
    *) err "architecture non supportée : $(uname -m)" ;;
  esac
  URL="https://github.com/$REPO/releases/latest/download/$asset"
  printf '  Téléchargement de %s…\n' "$asset"
  curl -fsSL "$URL" -o "$STAGE" || err "téléchargement échoué ($URL)"
fi

codesign --force --sign - "$STAGE" >/dev/null 2>&1 || true
chmod +x "$STAGE"
mv -f "$STAGE" "$DEST/$BIN"   # renommage atomique (évite "zsh: killed")
ok "installé : $DEST/$BIN"

case ":$PATH:" in
  *":$DEST:"*) ;;
  *) printf '\n  Ajoute à ~/.zshrc :\n    export PATH="%s:$PATH"\n' "$DEST" ;;
esac
printf '\n  Lance : \033[1m%s\033[0m\n' "$BIN"
