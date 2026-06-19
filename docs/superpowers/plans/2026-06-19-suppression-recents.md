# Suppression de l'onglet Récents — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Retirer entièrement l'onglet « ◷ Récents » et son code mort ; la page d'accueil devient ★ Favoris.

**Architecture:** Suppression d'une fonctionnalité TUI (bubbletea/lipgloss). Le code Récents est concentré dans `workspace.go` (données) et `view.go` (rendu) ; deux tests le référencent. En Go, retirer la valeur d'enum `KindRecents` oblige à retirer simultanément tous ses `case`/usages : le retrait du code de production est donc un bloc atomique unique, encadré par un test rouge→vert.

**Tech Stack:** Go 1.26, bubbletea, lipgloss.

## Global Constraints

- `gofmt -l .` → aucun fichier listé.
- `go vet ./...` → silencieux.
- `go build ./...` → ok.
- `go test ./...` → ok.
- Aucun changement d'UX, de raccourci ou de format de config (`~/.config/bagent/{favorites,workspaces}`).
- Branche de travail : `suppression-recents` (déjà créée, spec commitée dessus).

---

### Task 1 : Retirer l'onglet Récents (code + tests)

**Files:**
- Modify: `internal/app/view_test.go:21-33` (assertion d'ordre des pages)
- Modify: `internal/app/workspace.go` (enum l.13-17, `maxRecents` l.35, `resolveClaudePath` l.94-132, `loadRecents` l.134-186, `buildPages` l.194-198 et l.210)
- Modify: `internal/app/view.go` (`emptyMessage` l.90-99, `footerKeys` l.231-238)
- Modify: `internal/app/reorder_test.go:39-50` (model fictif)

**Interfaces:**
- Consumes: rien (tâche autonome).
- Produces: après cette tâche, `buildPages()` renvoie `[★ Favoris, …projets]` ; `KindRecents`, `maxRecents`, `resolveClaudePath`, `loadRecents` n'existent plus. `favorisIndex(pages []Page) int` est conservée et renvoie 0.

- [ ] **Step 1 : Adapter le test d'ordre des pages (le rendre rouge)**

Dans `internal/app/view_test.go`, remplacer le commentaire et l'assertion `pages[0]` :

```go
// buildPages : Favoris en tête, suivi d'un onglet par projet.
func TestBuildPages(t *testing.T) {
	pages := buildPages()
	if len(pages) < 1 {
		t.Fatalf("attendu >=1 page, obtenu %d", len(pages))
	}
	if pages[0].Kind != KindFavoris {
		t.Errorf("1re page doit être Favoris, obtenu %v", pages[0].Title)
	}
	if pages[favorisIndex(pages)].Kind != KindFavoris {
		t.Errorf("page Favoris introuvable")
	}
}
```

- [ ] **Step 2 : Lancer ce test pour confirmer l'échec**

Run: `go test ./internal/app/ -run TestBuildPages -v`
Expected: FAIL — `1re page doit être Favoris, obtenu Récents` (buildPages place encore Récents en tête).

- [ ] **Step 3 : Retirer le code de production Récents (workspace.go)**

Dans `internal/app/workspace.go` :

Enum — supprimer `KindRecents` :

```go
const (
	KindFavoris PageKind = iota
	KindProjet
)
```

Supprimer la constante (la ligne `const maxRecents = 8`).

Supprimer entièrement la fonction `resolveClaudePath` (commentaire + corps, l.94-132).

Supprimer entièrement la fonction `loadRecents` (commentaire + corps, l.134-186).

Dans `buildPages()`, supprimer le bloc de construction des récents :

```go
	// Récents (à gauche)
	var recItems []Item
	for _, r := range loadRecents(dirs) {
		recItems = append(recItems, Item{Name: filepath.Base(r), FullPath: r, Fav: favs[r]})
	}
```

et l'entrée Récents du slice `pages`, qui devient :

```go
	pages := []Page{
		{Title: "Favoris", Icon: "★", Kind: KindFavoris, Items: favItems},
	}
```

(la variable `dirs` reste utilisée plus bas pour les pages projet ; l'import `sort` reste utilisé par `loadSubdirs`.)

- [ ] **Step 4 : Retirer les cas Récents du rendu (view.go)**

Dans `internal/app/view.go`, `emptyMessage` devient :

```go
func emptyMessage(kind PageKind) string {
	switch kind {
	case KindFavoris:
		return "Aucun favori — a pour ajouter un chemin, f depuis un projet"
	default:
		return "Projet vide — a pour créer un dossier"
	}
}
```

Dans `footerKeys`, supprimer le `case KindRecents` du `switch page.Kind` (l.236-237). Le switch ne conserve que `KindFavoris` et `KindProjet`.

- [ ] **Step 5 : Adapter le model fictif de reorder (reorder_test.go)**

Dans `internal/app/reorder_test.go`, `reorderModel()` devient (retrait de la page Récents, `pageIdx` 2 → 1 pour rester sur « alpha ») :

```go
// reorderModel construit un model en mémoire (sans I/O) : Favoris, puis deux
// projets, le focus en haut sur le premier projet.
func reorderModel() model {
	m := model{mode: modeList, focus: focusBar, width: 100, height: 30}
	m.pages = []Page{
		{Title: "Favoris", Icon: "★", Kind: KindFavoris},
		{Title: "alpha", Kind: KindProjet, Parent: "/p/alpha"},
		{Title: "beta", Kind: KindProjet, Parent: "/p/beta"},
	}
	m.pageIdx = 1
	return m
}
```

- [ ] **Step 6 : Lancer toute la suite pour confirmer le vert**

Run: `go test ./...`
Expected: PASS (tous les paquets). En particulier `TestBuildPages` passe et les tests `reorder` restent verts avec `pageIdx = 1`.

- [ ] **Step 7 : Vérifier format / vet / build**

Run: `gofmt -l . && go vet ./... && go build ./...`
Expected: aucune sortie de `gofmt`, `go vet` silencieux, build ok.

- [ ] **Step 8 : Commit**

```bash
git add internal/app/workspace.go internal/app/view.go internal/app/view_test.go internal/app/reorder_test.go
git commit -m "Supprime l'onglet Récents

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

### Task 2 : Documentation

**Files:**
- Modify: `README.md:15` (puce Récents)
- Regenerate: `screenshot.png`

**Interfaces:**
- Consumes: l'app construite en Task 1 (page d'accueil ★ Favoris, pas d'onglet Récents).
- Produces: README et capture alignés sur le comportement final.

- [ ] **Step 1 : Retirer la puce Récents du README**

Dans `README.md`, supprimer la ligne :

```
- **◷ Récents** — les derniers projets ouverts dans Claude Code (`~/.claude/projects`).
```

Vérifier qu'aucune autre phrase du README ne décrit l'onglet Récents (recherche : `Récents`, `◷`).

- [ ] **Step 2 : Régénérer le screenshot**

`screenshot.png` (référencé en `README.md:7`) montre l'ancien onglet Récents. Le régénérer en lançant l'app et en capturant l'écran d'accueil (★ Favoris en tête), selon le même procédé que les captures précédentes. Cette étape est visuelle/manuelle ; si elle nécessite une action de l'utilisateur, le signaler avant de commiter.

- [ ] **Step 3 : Commit**

```bash
git add README.md screenshot.png
git commit -m "Doc : retire l'onglet Récents (README + screenshot)

Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>"
```

---

## Notes de revue (self-review)

- **Couverture spec :** workspace.go (enum, maxRecents, resolveClaudePath, loadRecents, buildPages) → Task 1 Step 3 ; view.go (emptyMessage, footerKeys) → Task 1 Step 4 ; view_test.go → Step 1 ; reorder_test.go → Step 5 ; favorisIndex conservée → Step 3 (non touchée) ; README → Task 2 Step 1 ; screenshot → Task 2 Step 2. Tous les points de la spec sont couverts.
- **Ordre TDD :** le test devient rouge (Step 1-2) avant le retrait du code (Step 3-5), vert ensuite (Step 6). Le retrait enum + cases + usages est groupé car aucun état intermédiaire ne compile en Go.
- **Cohérence des types :** `KindFavoris`/`KindProjet` conservent leurs valeurs iota (0,1) ; `favorisIndex` garde sa signature `func(pages []Page) int`.
