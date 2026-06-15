#!/bin/sh
# Installe bagent. Usage :
#   curl -fsSL https://raw.githubusercontent.com/damienp199/bagent/main/install.sh | sh
# ou, depuis un clone du repo :
#   ./install.sh
set -e

REPO="https://github.com/damienp199/bagent.git"
BIN="bagent"
DEST="${BAGENT_BIN:-$HOME/.local/bin}"

err() { printf '  \033[31m✗\033[0m %s\n' "$1" >&2; exit 1; }
ok()  { printf '  \033[32m✓\033[0m %s\n' "$1"; }

command -v go >/dev/null 2>&1 || err "Go est requis — https://go.dev/dl/"

# Construit depuis le repo courant si possible, sinon clone temporaire.
CLEAN=""
if [ -f "go.mod" ] && grep -q "damienp199/bagent" go.mod 2>/dev/null; then
  SRC="$(pwd)"
else
  command -v git >/dev/null 2>&1 || err "git est requis"
  SRC="$(mktemp -d)"
  CLEAN="$SRC"
  printf '  Clonage du dépôt…\n'
  git clone --depth 1 "$REPO" "$SRC" >/dev/null 2>&1 || err "échec du clone"
fi

printf '  Compilation…\n'
( cd "$SRC" && go build -o "$BIN" . ) || err "échec de la compilation"

mkdir -p "$DEST"
STAGE="$DEST/.$BIN.new"
cp -f "$SRC/$BIN" "$STAGE"

# macOS Apple Silicon : (re)signature ad-hoc pour éviter un "zsh: killed".
if [ "$(uname)" = "Darwin" ]; then
  codesign --force --sign - "$STAGE" >/dev/null 2>&1 || true
fi

chmod +x "$STAGE"
mv -f "$STAGE" "$DEST/$BIN"   # renommage atomique (même dossier → nouvel inode)
ok "installé : $DEST/$BIN"

[ -n "$CLEAN" ] && rm -rf "$CLEAN"

# Vérifie que DEST est dans le PATH.
case ":$PATH:" in
  *":$DEST:"*) ;;
  *)
    printf '\n  Ajoute %s à ton PATH (dans ~/.zshrc) :\n' "$DEST"
    printf '    export PATH="%s:$PATH"\n' "$DEST"
    ;;
esac

printf '\n  Terminé. Lance : \033[1m%s\033[0m\n' "$BIN"
