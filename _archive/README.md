# CLAUDE BOOT

Un lanceur TUI minimaliste pour [Claude Code](https://docs.anthropic.com/en/docs/claude-code).

Enregistrez vos dossiers de projets favoris et lancez Claude Code en une touche — fini les `cd` à répétition. Les projets Claude récemment utilisés s'affichent aussi automatiquement.

![screenshot](screenshot.png)

## Installation

```sh
curl -fsSL https://raw.githubusercontent.com/damienp199/bclaude/main/install.sh | sh
```

Nécessite le [CLI Claude Code](https://docs.anthropic.com/en/docs/claude-code).

Assurez-vous que `~/.local/bin` est dans votre PATH. Ajoutez ceci à votre `~/.zshrc` (ou `~/.bashrc`) si nécessaire :

```sh
export PATH="$HOME/.local/bin:$PATH"
```

## Utilisation

```sh
bclaude
# ou plus court :
bcl
```

| Touche | Action |
|--------|--------|
| `↑` `↓` / `j` `k` | Naviguer |
| `Enter` | Ouvrir le workspace dans Claude Code |
| `1`–`9` | Ouvrir directement le workspace correspondant |
| `a` | Ajouter un workspace (ou épingler un récent) |
| `s` | Supprimer un workspace |
| `d` | Définir par défaut |
| `q` | Quitter |

### Récents

Le menu affiche automatiquement jusqu'à 5 projets Claude Code récemment utilisés sous vos workspaces épinglés. Ils sont détectés depuis `~/.claude/projects/` et triés par dernière utilisation. Les dossiers supprimés ou déjà épinglés sont exclus.

Appuyez sur `a` sur un récent pour l'épingler, ou `d` pour le définir par défaut.

### Ajouter un workspace

Appuyez sur `a` dans le menu, puis saisissez le chemin complet de votre dossier de projet. Deux modes sont proposés :

- **Direct (`d`)** — Ajoute le dossier comme un projet unique
- **Parent (`p`)** — Ajoute un dossier parent dont tous les sous-dossiers sont listés dynamiquement comme autant de projets

Le mode parent est utile quand vous organisez vos projets dans un dossier commun (ex : `~/Documents/Business/` contenant `Projet A`, `Projet B`, etc.). Les sous-dossiers apparaissent automatiquement sous un header de section — pas besoin de les ajouter un par un.

Vous pouvez aussi éditer manuellement `~/.config/bclaude/workspaces` (un chemin par ligne) :

```sh
# Workspace direct
echo "/path/to/project" >> ~/.config/bclaude/workspaces

# Dossier parent (préfixe >)
echo ">/path/to/parent" >> ~/.config/bclaude/workspaces
```

### Lancer le défaut directement

```sh
bclaude -d
```

Ouvre immédiatement le workspace par défaut (marqué `●`) sans afficher le menu.

## Fonctionnement

- Les workspaces sont stockés dans `~/.config/bclaude/workspaces` (un chemin par ligne, préfixe `>` pour les parents)
- Les projets récents sont lus depuis `~/.claude/projects/` (créé par Claude Code)
- Sélectionner un workspace ou un récent fait un `cd` et lance `claude`
- Aucune dépendance en dehors de `zsh` et `claude`

## Désinstallation

```sh
rm ~/.local/bin/bclaude ~/.local/bin/bcl
rm -rf ~/.config/bclaude
```
