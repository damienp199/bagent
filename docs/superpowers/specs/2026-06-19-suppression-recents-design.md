# Suppression de l'onglet Récents

Date : 2026-06-19
Statut : proposé (en attente de relecture)

## Contexte & motivation

L'onglet « ◷ Récents » lit uniquement `~/.claude/projects/` et reconstruit le chemin
réel de chaque projet en décodant le nom de dossier encodé par Claude
(`resolveClaudePath`). Deux problèmes le rendent inutile en pratique :

1. **Quasi vide.** `loadRecents` retire tout récent situé sous un workspace/groupe
   épinglé. Comme la quasi-totalité des projets vivent sous des groupes épinglés
   (`>…/Dev`, `>…/Apps`, `>…/AgenticOS/WORKSPACES`), ils sont tous masqués — il ne
   reste qu'un projet hors groupe (« Downloads »).
2. **Source unique et obsolète.** bagent lance désormais aussi Codex et VSCode. Un
   onglet « Récents » alimenté par le seul historique Claude est trompeur :
   incomplet par construction.

Plutôt qu'étendre les récents à un agrégat multi-outils (Codex via
`~/.codex/sessions/**/rollout-*.jsonl`, VSCode via `storage.json`), la décision
retenue est la **suppression pure et simple** de la fonctionnalité. Les groupes
épinglés couvrent déjà l'ensemble des projets, et les favoris couvrent l'accès
rapide ; les récents sont redondants et coûteux à fiabiliser.

## Objectif

Retirer entièrement l'onglet Récents et le code associé. La page d'accueil devient
**★ Favoris**. Aucun autre changement d'UX ni de raccourci.

## Comportement résultant

Onglets : **★ Favoris** (accueil) puis un onglet par projet/groupe. Les raccourcis,
les fichiers de config (`~/.config/bagent/{favorites,workspaces}`) et le reste de
l'UX sont inchangés.

## Périmètre — points de couplage à retirer

### `internal/app/workspace.go`

- Retirer `KindRecents` de l'enum `PageKind` (l.16). Retirer la dernière valeur de
  l'`iota` ne décale pas `KindFavoris` (0) ni `KindProjet` (1).
- Retirer `const maxRecents = 8` (l.35).
- Retirer `resolveClaudePath()` (l.94-132) — aucun autre appelant.
- Retirer `loadRecents()` (l.134-186) — aucun autre appelant.
- Dans `buildPages()` : retirer le bloc de construction `recItems` (l.194-198) et
  l'entrée `{Title: "Récents", … KindRecents}` du slice `pages` (l.210). Favoris
  reste la première page.
- **Conserver** `favorisIndex()` tel quel : il renverra 0, ce qui évite de coder
  l'ordre des pages en dur dans `tui.go:82` et `app.go:57`. Choix défensif.
- L'import `sort` reste nécessaire (`loadSubdirs`).

### `internal/app/view.go`

- `emptyMessage` : retirer le `case KindRecents` (l.94-95).
- `footerKeys` : retirer le `case KindRecents` (l.236-237).

### `internal/app/view_test.go`

- `TestBuildPages` (l.21-33) : la première page devient `KindFavoris`. Adapter
  l'assertion et le commentaire (« Favoris en tête »).

### `internal/app/reorder_test.go`

- `reorderModel()` (l.39-51) : retirer la page fictive `{… KindRecents}` (l.44).
  Les pages deviennent `[Favoris, alpha, beta]`, donc `m.pageIdx` passe de **2 à 1**
  pour rester positionné sur le projet « alpha ». Adapter le commentaire (l.39-40).

### `README.md`

- Retirer la puce « **◷ Récents** — les derniers projets ouverts dans Claude Code »
  (l.15).

### `screenshot.png`

- L'image affiche l'onglet Récents. À régénérer en fin de travail (étape finale,
  hors compilation).

## Vérification

À la fin :

1. `gofmt -l .` → aucun fichier listé
2. `go vet ./...` → silencieux
3. `go build ./...` → ok
4. `go test ./...` → ok
5. Lancement manuel : l'app ouvre sur ★ Favoris, aucun onglet Récents, navigation
   et raccourcis intacts.
6. Régénérer `screenshot.png`.

## Non inclus

- Tout agrégat multi-outils de récents (Codex / VSCode). Explicitement écarté.
- Toute autre modification d'UX, de raccourci ou de format de config.
