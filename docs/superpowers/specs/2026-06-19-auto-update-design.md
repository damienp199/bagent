# Auto-update bagent — design

Date : 2026-06-19
Statut : validé (brainstorming)

## Objectif

Permettre aux clients de bagent de mettre à jour le binaire sans connaître la
commande d'installation, et de savoir qu'une nouvelle version existe.

Périmètre retenu (niveau « commande + notif ») :

- une commande `bagent --update` qui met à jour le binaire depuis la dernière
  release GitHub ;
- une notification discrète dans le TUI quand une version plus récente est
  disponible.

**Hors périmètre :** remplacement silencieux automatique du binaire au
lancement (self-update à la Claude Code). Écarté pour garder le contrôle côté
utilisateur et limiter la surface d'attaque.

## Contexte

- Binaire Go (TUI Bubble Tea), macOS uniquement, distribué via GitHub Releases.
- `install.sh` télécharge le binaire `latest`, le re-signe ad-hoc et l'installe
  par renommage atomique (`mv -f`), ce qui évite le `zsh: killed` dû au cache de
  signature AMFI du noyau. C'est la **source unique de vérité** pour
  l'installation : l'auto-update la réutilise, ne la réimplémente pas.
- Module Go : `github.com/damienp199/bagent`, package applicatif
  `internal/app`.

## Architecture

### 1. Version embarquée

Variable de package dans `internal/app` :

```go
var version = "dev"
```

Injectée au build via ldflags :

```sh
go build -ldflags "-X github.com/damienp199/bagent/internal/app.version=v0.1.3" -o ...
```

- La procédure de release (compilation des deux cibles macOS) passe désormais ce
  flag avec le tag publié.
- `install.sh` en mode dev (compilation locale) ne passe pas de version : elle
  reste `"dev"`, ce qui **désactive la notif** pour les builds locaux.
- La mémoire `bagent-release-binaire` est mise à jour pour inclure le ldflags.

### 2. Module `internal/app/update.go`

Unité isolée, testable sans réseau (fonction de fetch injectable).

- `latestRelease(fetch func(string) ([]byte, error)) (string, error)` — appelle
  `https://api.github.com/repos/damienp199/bagent/releases/latest`, extrait
  `tag_name` du JSON. Timeout réseau 3 s côté client HTTP réel.
- **Cache** : fichier `~/.config/bagent/.update-check`, JSON
  `{"checked_at": <unix>, "tag": "v0.1.3"}`. Si l'horodatage a moins de 24 h, on
  relit le cache sans appel réseau. Sinon on interroge l'API et on réécrit le
  cache.
- `isNewer(latest, current string) bool` — comparaison semver simple : découpe
  `vMAJOR.MINOR.PATCH`, compare numériquement. Renvoie `false` si
  `current == "dev"`, si un tag est non parsable, ou si `latest <= current`.
  Garantit l'absence de fausse notif (build local en avance sur la release).
- `checkForUpdate()` — orchestre cache + `latestRelease` + `isNewer`, renvoie le
  tag à proposer (`""` si rien). Toute erreur (réseau, hors-ligne, parse) est
  avalée silencieusement : `""`.

### 3. Intégration TUI (notif footer)

- `model` gagne un champ `updateTag string` (vide = pas de notif).
- `Init()` retourne, en plus de `refreshCmd()`, une `tea.Cmd` asynchrone
  `updateCheckCmd()` qui exécute `checkForUpdate()` hors du thread de rendu et
  renvoie `updateAvailableMsg{tag string}`. Le TUI s'ouvre instantanément.
- `Update` gère `updateAvailableMsg` : `m.updateTag = msg.tag`.
- `footer()` : si `updateTag != ""` **et** qu'aucun `status` temporaire n'est
  actif, la ligne du bas affiche `● <tag> dispo · bagent --update` (style orange
  discret). Le `status` ponctuel (✓/✗) reste prioritaire et masque la notif le
  temps de son affichage (statusDelay), puis la notif réapparaît.

La notif n'apparaît jamais en mode `modeInput` / `modeDelConfirm` / `modeReorder`
(le footer y affiche déjà un contenu dédié — comportement inchangé).

### 4. Commande `bagent --update` et `--version`

Dans `Run()` (app.go), avant le lancement du TUI :

- `case "--update", "-u":` — exécute `curl -fsSL
  https://raw.githubusercontent.com/damienp199/bagent/main/install.sh | sh` via
  `os/exec`, en transmettant stdout/stderr au terminal. Réutilise donc tout le
  comportement d'`install.sh` (download + re-signature + mv atomique).
- `case "--version", "-v":` — affiche `version` et termine.

Mise à jour de `printHelp()` pour documenter `--update` et `--version`.

## Flux de données

```
Lancement TUI
  └─ Init() ─┬─ refreshCmd()        (existant)
             └─ updateCheckCmd() ──► checkForUpdate()
                                       ├─ cache < 24h ? ─► tag du cache
                                       └─ sinon ─► API GitHub ─► écrit cache
                                     ──► updateAvailableMsg{tag}
                                          └─ model.updateTag = tag
                                               └─ footer() affiche la notif

bagent --update
  └─ exec( curl install.sh | sh ) ─► download + codesign + mv atomique
```

## Gestion des erreurs

- Réseau indisponible / timeout / HTTP non-2xx / JSON invalide → `checkForUpdate`
  renvoie `""` : pas de notif, aucun message d'erreur visible.
- Cache illisible / corrompu → traité comme absent (on ré-interroge l'API).
- `--update` : la sortie d'`install.sh` (succès ou échec) est affichée telle
  quelle ; le code de sortie est propagé.

## Tests (`internal/app/update_test.go`)

- `latestRelease` : parsing du `tag_name` depuis un payload JSON fixture (fetch
  injecté) ; erreur de fetch propagée.
- `isNewer` : cas plus récent, égal, plus ancien, `current == "dev"`, tags non
  parsables.
- Cache : écriture puis relecture ; expiration au-delà de 24 h (horodatage
  injecté) ; fichier corrompu ⇒ absent.
- Footer : rendu contient la notif quand `updateTag` est défini et qu'aucun
  status n'est actif ; status prioritaire masque la notif.

Aucun test n'effectue d'appel réseau réel (fetch et horloge injectés).

## Décisions de conception

- **Comparaison semver** plutôt qu'égalité de tag : évite de notifier un build
  local en avance sur la release.
- **Coexistence notif / status** : le status ponctuel a priorité sur la ligne du
  footer ; la notif réapparaît ensuite.
- **`--update` réutilise `install.sh`** : une seule implémentation du piège
  `zsh: killed`, pas de duplication de la logique de téléchargement en Go.
- **Cache 24 h** : respecte la limite d'API GitHub non authentifiée (60 req/h) et
  évite un appel réseau systématique.
