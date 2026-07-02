# bagent — extension Raycast

Ouvre un workspace bagent depuis Raycast, sans passer par le TUI.

La liste est lue **en direct** via `bagent json` : toute modif faite dans le TUI
(favoris, projets) apparaît aussitôt.

## Commande

**bagent** — liste tous tes workspaces (favoris marqués `★`, filtrables par nom
ou groupe). Actions sur l'item sélectionné :

| Touche | Ouvre |
|---|---|
| `⏎` | VSCode |
| `⌘⏎` | Claude Code |
| `⌥⏎` | Codex |
| `⌘F` | Finder |
| `⌘.` | copier le chemin |

Claude Code et Codex s'ouvrent dans le terminal choisi dans les préférences de
l'extension (Terminal.app ou iTerm).

## Prérequis

Cette extension pilote le **CLI bagent**. Installe-le d'abord (macOS) :

```sh
curl -fsSL https://raw.githubusercontent.com/damienp199/bagent/main/install.sh | sh
```

Le binaire atterrit dans `~/.local/bin/bagent` (emplacement par défaut lu par
l'extension). Si tu l'installes ailleurs, renseigne son chemin dans les
préférences de l'extension. Sans bagent, la commande affiche un lien
d'installation.

## Installation (dev local)

```sh
cd launchers/raycast
npm install
npm run dev     # ouvre Raycast en mode développement, commande dispo aussitôt
```

`npm run dev` importe l'extension dans ton Raycast et la garde en watch. Pour
figer une version installée : `npm run build`.

## Préférences

- **Terminal (Claude/Codex)** : `Terminal.app` (défaut) ou `iTerm`.
- **Binaire bagent** : chemin custom si `bagent` n'est pas dans `~/.local/bin`.
