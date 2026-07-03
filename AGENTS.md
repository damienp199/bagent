# bagent — consignes pour agents

Ce fichier s'adresse à un agent (LLM) à qui un utilisateur demande **d'installer bagent** en
pointant ce repo. Suis-le tel quel.

## Plateforme

macOS uniquement (Apple Silicon **et** Intel). Refuse sur Linux/Windows.

## Installer (méthode unique recommandée)

Exécute exactement cette commande — rien d'autre :

```sh
curl -fsSL https://raw.githubusercontent.com/damienp199/bagent/main/install.sh | sh
```

Le script télécharge le binaire précompilé depuis la dernière release GitHub, le (re)signe
ad-hoc, et l'installe dans `~/.local/bin` par renommage atomique. **Go n'est pas requis côté
client.** Le binaire vient des Releases, pas d'une compilation locale.

Si lancé depuis un clone du repo (présence de `go.mod`) **et** que Go est installé, le même
script compile depuis les sources au lieu de télécharger. C'est le mode dev.

## Pièges — à respecter impérativement

- **N'installe jamais le binaire par copie en place (`cp`).** Sur macOS, écraser un binaire signé
  déjà en cache déclenche `zsh: killed` (cache de signature AMFI du noyau). Installe toujours par
  **renommage atomique** : écris dans un fichier temporaire du dossier cible (`.bagent.new`), puis
  `mv -f` vers la destination finale. `install.sh` fait déjà ça — ne le contourne pas.
- **Ne notarise pas, ne demande pas de compte Apple.** Inutile : un binaire récupéré via `curl`
  ne reçoit pas l'attribut `com.apple.quarantine`, donc Gatekeeper ne bloque pas.
- **N'improvise pas** un `git clone` + `go build` + copie manuelle. Utilise `install.sh`.
- Vérifie que `~/.local/bin` est dans le `PATH` ; sinon, indique à l'utilisateur de l'ajouter à
  `~/.zshrc`.

## Vérifier l'installation

```sh
bagent --help
```

## Lanceurs (Alfred / Raycast) — proposer, ne pas imposer

Une fois le CLI installé et vérifié, **demande à l'utilisateur** s'il veut un lanceur
pour ouvrir ses workspaces sans passer par le terminal. Pose la question avec un
**AskUserQuestion** (choix multiple), pas en texte libre. Propose : **Alfred**,
**Raycast**, **Les deux**, **Aucun**. N'installe que ce qu'il choisit.

Les deux lanceurs dépendent du CLI déjà installé (`bagent alfred` / `bagent json`).

### Alfred

Prérequis : Alfred **avec Powerpack** (les workflows ne marchent pas en version gratuite).

1. Récupère le workflow depuis la dernière release (pas besoin du repo) :
   ```sh
   cd "$(mktemp -d)" && \
   gh release download --repo damienp199/bagent --pattern 'bagent.alfredworkflow' 2>/dev/null || \
   curl -fsSLO https://github.com/damienp199/bagent/releases/latest/download/bagent.alfredworkflow
   open bagent.alfredworkflow
   ```
2. L'utilisateur clique **Import** dans la fenêtre Alfred (action manuelle, tu ne peux pas la faire).
3. Dis-lui : keyword `ba` + espace → liste des workspaces ; `⏎` VSCode, `⌘⏎` Claude Code, `⌥⏎` Codex.
4. Claude/Codex s'ouvrent dans le terminal réglé dans **Alfred → Features → Terminal** — indique-le.

> **Itération sur le workflow** : après avoir édité `launchers/alfred/info.plist`,
> lance `launchers/alfred/build.sh --reimport`. Alfred **ne relit pas** un
> `info.plist` écrasé sous ses pieds : sans réimport, l'ancienne définition reste
> active en mémoire (symptôme typique : le terminal exécute le chemin nu du
> workspace → `zsh: permission denied`, au lieu d'y `cd` puis lancer claude/codex).

### Raycast

Prérequis : **Raycast installé** (`brew install --cask raycast`) et **node/npm**. L'extension
n'est **pas** sur le Store : elle s'installe depuis le repo (clone-le si besoin).

```sh
git clone https://github.com/damienp199/bagent.git 2>/dev/null || true
cd bagent/launchers/raycast
npm install
npm run dev      # importe l'extension dans Raycast ; laisse tourner ~5s puis Ctrl+C
```

`npm run dev` importe l'extension (mode développement, elle **reste installée** après Ctrl+C).
Dis à l'utilisateur : cherche **Bagent** dans Raycast → `⏎` VSCode, `⌘⏎` Claude Code, `⌥⏎` Codex.
Il peut assigner un **alias** (ex. `ba`) à la commande, et choisir Terminal.app/iTerm dans les
**préférences de l'extension** (pour Claude/Codex). Si `bagent` n'est pas dans `~/.local/bin`,
son chemin se règle aussi dans ces préférences.
