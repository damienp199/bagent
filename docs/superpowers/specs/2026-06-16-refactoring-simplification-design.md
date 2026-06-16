# Refactoring — simplification & standardisation de bagent

Date : 2026-06-16
Statut : proposé (en attente de relecture)

## Objectif

L'application est terminée et fonctionnelle. Profiter de ce moment pour nettoyer le
code interne : supprimer les duplications, standardiser les patterns récurrents,
retirer le code rendu inutile par la version de Go. **Aucun changement de
comportement** : l'UX, les raccourcis et les fichiers de config restent identiques.
Les tests existants (couverture 55 %, centrée sur `workspace.go` et le rendu) servent
de filet de sécurité.

Profondeur retenue : **modérée**. Pas de refonte des frontières de fichiers/modules,
pas de renommage de symboles publics, pas de découpage de `tui.go`.

## Périmètre

5 fichiers source (~1200 lignes) : `app.go`, `tui.go`, `view.go`, `workspace.go`,
`launch.go`.

## Contrainte transverse

À chaque étape : `gofmt`, `go vet ./...`, `go build ./...`, `go test ./...` doivent
rester verts. Les tests ne sont pas modifiés (sauf si un refactoring change une
signature qu'ils appellent — voir chaque item).

## Les refactorings

### R1 — Helpers de construction des messages de statut

**Problème.** Le pattern `stRed.Render("✗") + " " + msg` (et ses variantes `✓`/`★`)
est répété ~12 fois dans `tui.go` (`launch`, `applyInput`, `updateBar`,
`updateItems`, `updateDelConfirm`).

**Solution.** Trois helpers, placés dans `view.go` (ils relèvent du rendu et
utilisent les styles déjà définis là) :

```go
func statusErr(msg string) string { return stRed.Render("✗") + " " + msg }
func statusOK(msg string) string  { return stGreen.Render("✓") + " " + msg }
func statusFav(msg string) string { return stFav.Render("★") + " " + msg }
```

Les appels deviennent p.ex. `m.status = statusOK("VSCode " + stDim.Render(it.Name))`.
Le `stDim.Render(name)` reste au site d'appel (varie selon le contexte).

Impact tests : aucun (les chaînes rendues sont identiques).

### R2 — Helper `setStatus` (statut + commande d'effacement)

**Problème.** Presque chaque action fait deux lignes couplées :
`m.status = ... ; return m, clearStatusCmd()`.

**Solution.** Un helper sur le modèle :

```go
func (m *model) setStatus(s string) tea.Cmd {
    m.status = s
    return clearStatusCmd()
}
```

Les retours deviennent `return m, m.setStatus(statusOK(...))`. `m` étant un paramètre
valeur adressable dans les méthodes `Update`, l'appel à récepteur pointeur est légal.

Combiné à R1, une action passe de 2–3 lignes à 1.

Impact tests : aucun (comportement identique : statut posé + tick d'effacement).

### R3 — Helper `removeLine` pour les mutations de listes

**Problème.** `removeEntry`, `removeFavorite` et `toggleFavorite` (branche de retrait)
réimplémentent chacun le filtrage « retirer une ligne d'une slice ».

**Solution.** Un helper pur dans `workspace.go` :

```go
func removeLine(lines []string, entry string) []string {
    out := make([]string, 0, len(lines))
    for _, l := range lines {
        if l != entry {
            out = append(out, l)
        }
    }
    return out
}
```

`removeEntry`, `removeFavorite` et la branche retrait de `toggleFavorite` l'utilisent.
Note : les versions actuelles renvoient une slice `nil` quand le résultat est vide,
`removeLine` renvoie une slice vide non-nil — `writeLines` traite les deux de façon
identique (jointure → `""`). Comportement inchangé, validé par `TestFavorisRemoveWithF`
et `TestPruneDeadEntries`.

Impact tests : aucun.

### R4 — Constantes nommées pour les chaînes magiques

**Problème.** Les actions de saisie (`"favPath"`, `"newProjet"`, `"newDir"`) et les
actions de lancement (`"vscode"`, `"claude"`, `"codex"`) circulent en `string` brut
entre `tui.go`, `view.go`, `app.go` et `launch.go`.

**Solution.** Des constantes `string` nommées (pas de nouveaux types, pour éviter la
conversion aux frontières d'exec où `"claude"`/`"codex"` servent aussi de nom de
binaire) :

```go
// actions de saisie (tui.go)
const (
    inFavPath   = "favPath"
    inNewProjet = "newProjet"
    inNewDir    = "newDir"
)

// actions de lancement (app.go)
const (
    actVSCode = "vscode"
    actClaude = "claude"
    actCodex  = "codex"
)
```

Remplacement de tous les littéraux correspondants dans les `switch` et appels.

**Tradeoff assumé.** On garde le type `string` (constantes nommées) plutôt que des
types dédiés : `actClaude`/`actCodex` sont passés tels quels à `toolAvailable` et
`runInTerminal` comme noms de binaire ; un type dédié imposerait des conversions sans
gain réel à cette échelle.

Impact tests : aucun (valeurs identiques).

### R5 — Suppression du `min` maison

**Problème.** `view.go` définit `func min(a, b int) int`. Go 1.26 (cf. `go.mod`)
fournit `min`/`max` natifs.

**Solution.** Supprimer la fonction ; l'unique appel `min(width-4, 60)` utilise le
`min` natif sans changement de syntaxe.

Impact tests : aucun.

### R6 (optionnel) — `slices.Contains` à la place de `contains`

`workspace.go` définit `contains(lines, entry)`. `slices.Contains` (stdlib, dispo en
Go 1.26) fait la même chose. Remplacement possible et suppression du helper.

À trancher à la relecture : gain marginal, mais cohérent avec « standardiser ». Inclus
seulement si tu le valides.

## Non inclus (hors périmètre modéré)

- Découpage de `tui.go` (455 lignes) en sous-fichiers.
- Renommage de symboles publics (`Page`, `Item`, `PageKind`, etc.).
- Refonte de la couche persistance (format des fichiers `workspaces`/`favorites`).
- Toute modification d'UX, de raccourcis ou de comportement observable.

## Vérification

Après chaque commit et à la fin :

1. `gofmt -l .` → aucun fichier listé
2. `go vet ./...` → silencieux
3. `go build ./...` → ok
4. `go test ./...` → ok
5. Lancement manuel rapide de l'app pour confirmer le rendu (filet ultime).

## Découpage en commits proposé

1. R5 (min) + R6 (contains) — nettoyages stdlib, triviaux
2. R3 (removeLine) — dédup persistance
3. R1 (helpers statut) — dédup rendu
4. R2 (setStatus) — dédup contrôleur (dépend de R1)
5. R4 (constantes nommées) — standardisation transverse

Chaque commit est indépendamment vert et reviewable.
