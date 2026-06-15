#!/bin/sh
# bclaude installer
# Usage: curl -fsSL https://raw.githubusercontent.com/damienp199/bclaude/main/install.sh | sh
set -e

REPO="https://raw.githubusercontent.com/damienp199/bclaude/main/bclaude"
INSTALL_DIR="$HOME/.local/bin"
INSTALL_PATH="$INSTALL_DIR/bclaude"

# Couleurs
RED='\033[0;31m'
GREEN='\033[0;32m'
DIM='\033[2m'
RST='\033[0m'

echo ""
echo "  Installing bclaude..."
echo ""

# Vérifier que claude est installé
if ! command -v claude >/dev/null 2>&1; then
  printf "  ${RED}!${RST} Claude Code CLI introuvable.\n"
  printf "  ${DIM}Installe-le d'abord : https://docs.anthropic.com/en/docs/claude-code${RST}\n\n"
  exit 1
fi

# Créer le dossier si nécessaire
mkdir -p "$INSTALL_DIR"

# Télécharger
if command -v curl >/dev/null 2>&1; then
  curl -fsSL "$REPO" -o "$INSTALL_PATH"
elif command -v wget >/dev/null 2>&1; then
  wget -qO "$INSTALL_PATH" "$REPO"
else
  printf "  ${RED}!${RST} curl ou wget requis.\n"
  exit 1
fi

chmod +x "$INSTALL_PATH"

# Créer l'alias court bcl → bclaude
ln -sf "$INSTALL_PATH" "$INSTALL_DIR/bcl"

# Vérifier que le PATH contient ~/.local/bin
case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *)
    printf "  ${RED}!${RST} $INSTALL_DIR n'est pas dans ton PATH.\n"
    echo "  Ajoute cette ligne a ton ~/.zshrc (ou ~/.bashrc) :"
    echo ""
    echo "    export PATH=\"\$HOME/.local/bin:\$PATH\""
    echo ""
    ;;
esac

printf "  ${GREEN}Done.${RST} Lance ${GREEN}bclaude${RST} (ou ${GREEN}bcl${RST}) pour commencer.\n\n"
