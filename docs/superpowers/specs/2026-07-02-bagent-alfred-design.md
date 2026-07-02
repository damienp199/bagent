# bagent — intégration Alfred

## Objectif

Lancer un workspace depuis la barre Alfred (au lieu d'ouvrir le TUI dans un
terminal), pour diminuer la friction. Une seule source de vérité : les fichiers
de config existants (`~/.config/bagent/workspaces`, `favorites`), lus en direct.

## Principe

Alfred ne stocke pas la liste. Un Script Filter appelle `bagent alfred` à chaque
frappe ; le binaire relit la config et émet le JSON. Toute modif via le TUI (ou
édition manuelle des fichiers) apparaît immédiatement, sans réimporter le
workflow. On ne réimporte le `.alfredworkflow` que si sa structure change
(keyword, actions, format JSON).

## Livrables

### 1. `bagent alfred` (nouvelle sous-commande Go)

- Réutilise `buildPages()` pour résoudre favoris + sous-dossiers de projets.
- Aplatit en une liste unique (Alfred est plat), dédup par chemin complet.
- Ordre : favoris d'abord, puis les projets dans l'ordre du fichier. Alfred
  applique ensuite son tri par fréquence (via `uid`).
- Imprime le JSON Script Filter sur stdout. Aucune logique de lancement.

Format par item :

```json
{
  "uid": "/Users/x/Documents/Dev/bagent",
  "title": "[DEV] bagent",
  "subtitle": "★ ~/Documents/Dev/bagent",
  "arg": "/Users/x/Documents/Dev/bagent",
  "match": "bagent dev",
  "icon": { "type": "fileicon", "path": "/Users/x/Documents/Dev/bagent" },
  "mods": {
    "cmd": { "subtitle": "Ouvrir dans Claude Code" },
    "alt": { "subtitle": "Ouvrir dans Codex" }
  }
}
```

Règles d'affichage :

- `title` = `[GROUPE] nom`, où GROUPE = `strings.ToUpper(filepath.Base(filepath.Dir(chemin)))`.
- `subtitle` = chemin abrégé avec `~`, préfixé `★ ` si le chemin est un favori.
- `match` = nom + groupe (en minuscules) → `ba` ou `dev` matchent tous deux.
- `icon.type` = `fileicon` → vraie icône macOS du dossier.
- État vide (aucun workspace) → un item unique `valid:false`,
  title « Aucun workspace configuré », subtitle « Configure via bagent ».

### 2. `bagent open --tool=code <chemin>` (nouvelle sous-commande Go)

Colle pour ouvrir VSCode depuis Alfred. Réutilise `openVSCode` + `ensurePATH`
(indispensable : `code` n'est pas dans le PATH non-interactif d'Alfred). Ne gère
que `code` ; claude/codex passent par Alfred (voir ci-dessous).

### 3. `bagent.alfredworkflow` (fichier généré)

Prêt à double-cliquer. Contenu :

- Un Script Filter, keyword `ba`, **Argument Optional** (taper `ba␣` liste tout),
  script `bagent alfred`.
- Trois connexions depuis le Script Filter :
  - **⏎ (défaut)** → Run Script `bagent open --tool=code "$1"` → VSCode.
  - **⌘⏎ (command)** → Terminal Command `cd "$1" && exec claude` → Claude Code.
  - **⌥⏎ (option)** → Terminal Command `cd "$1" && exec codex` → Codex.

Pourquoi claude/codex via *Terminal Command* natif d'Alfred et pas via Go :
`runInTerminal` fait un `syscall.Exec` qui remplace le process courant dans un
terminal existant — inexploitable sans TTY depuis Alfred. L'objet Terminal
Command d'Alfred ouvre le terminal configuré par l'utilisateur, où le PATH du
`.zshrc` se charge naturellement (claude/codex trouvés sans `ensurePATH`).

## Câblage des touches

| Touche | Ouvre | Mécanisme |
|---|---|---|
| ⏎ | VSCode | Run Script → `bagent open --tool=code` |
| ⌘⏎ | Claude Code | Terminal Command → `cd … && exec claude` |
| ⌥⏎ | Codex | Terminal Command → `cd … && exec codex` |

## Hors périmètre

- Le CRUD (créer/renommer/supprimer/réordonner) reste dans le TUI bagent.
- Pas de spawn de terminal en Go (osascript, choix Terminal/iTerm) : délégué au
  Terminal Command d'Alfred.
