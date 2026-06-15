# bclaude

Application TUI en Zsh pour lancer Claude Code depuis des workspaces épinglés.

## Push

Quand l'utilisateur dit "push" :

1. **Screenshot** : Chercher le fichier image le plus récent sur le Bureau (`~/Desktop/`). S'il existe, le copier dans le projet en tant que `screenshot.png` (écrase l'ancien), puis supprimer l'original du Bureau.
2. **README** : Mettre à jour `README.md` si les changements le justifient (nouvelles fonctionnalités, nouvelles touches, changements de comportement). Ne pas toucher le README si les changements sont purement internes (refactor, bugfix mineur).
3. **Commit** : Committer tous les fichiers modifiés avec un message descriptif.
4. **Push** : Pousser sur `origin main`.

## Architecture

- `bclaude` : Script Zsh unique (~470 lignes), TUI avec écran alternatif
- `~/.config/bclaude/workspaces` : Fichier de config, un chemin par ligne
  - Ligne normale : workspace direct
  - Ligne préfixée `>` : dossier parent (liste dynamiquement ses sous-dossiers)
- Pas de dépendances externes (juste `zsh` + `claude`)
