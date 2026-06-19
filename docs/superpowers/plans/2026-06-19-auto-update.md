# Auto-update bagent Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ajouter une commande `bagent --update` et une notification discrète dans le TUI quand une version plus récente est disponible.

**Architecture:** Un module isolé `internal/app/update.go` interroge l'API GitHub des releases (avec cache 24 h et fetch injectable), compare en semver à la version embarquée, et expose le tag à proposer. Le TUI lance ce check en arrière-plan au démarrage et affiche le résultat dans le footer. `bagent --update` réutilise `install.sh`.

**Tech Stack:** Go, Bubble Tea (charmbracelet), bibliothèque standard (`net/http`, `encoding/json`, `os/exec`).

## Global Constraints

- macOS uniquement (déjà garanti en amont).
- Module Go : `github.com/damienp199/bagent`, package `internal/app`.
- Version embarquée injectée via ldflags `-X github.com/damienp199/bagent/internal/app.version=<tag>` ; défaut `"dev"`.
- Dossier de config réutilisé : `configDir()` (existant dans `workspace.go`) → `~/.config/bagent`.
- `install.sh` est la source unique de vérité pour l'installation (download + re-signature ad-hoc + `mv` atomique). `--update` l'exécute, ne le réimplémente pas.
- Aucun test n'effectue d'appel réseau réel : `fetch` et l'horodatage `now` sont injectés.
- Le check ne doit jamais produire de fausse notif : `isNewer` renvoie `false` si la version courante est `"dev"`, si un tag est non parsable, ou si `latest <= current`.

---

### Task 1 : Version embarquée + comparaison semver

**Files:**
- Create: `internal/app/update.go`
- Test: `internal/app/update_test.go`

**Interfaces:**
- Produces:
  - `var version = "dev"` — version embarquée du binaire.
  - `parseSemver(tag string) ([3]int, bool)` — découpe `vMAJOR.MINOR.PATCH`, `ok=false` si non parsable.
  - `isNewer(latest, current string) bool` — `true` ssi `latest` est strictement plus récent que `current` en semver, hors cas `dev`/non parsable.

- [ ] **Step 1 : Écrire les tests qui échouent**

```go
package app

import "testing"

func TestParseSemver(t *testing.T) {
	got, ok := parseSemver("v0.1.2")
	if !ok || got != [3]int{0, 1, 2} {
		t.Fatalf("parseSemver(v0.1.2) = %v,%v ; veut [0 1 2],true", got, ok)
	}
	if _, ok := parseSemver("v1.2"); ok {
		t.Fatal("parseSemver(v1.2) devrait échouer (3 segments requis)")
	}
	if _, ok := parseSemver("vX.Y.Z"); ok {
		t.Fatal("parseSemver(vX.Y.Z) devrait échouer (non numérique)")
	}
}

func TestIsNewer(t *testing.T) {
	cases := []struct {
		latest, current string
		want            bool
	}{
		{"v0.1.3", "v0.1.2", true},
		{"v0.2.0", "v0.1.9", true},
		{"v1.0.0", "v0.9.9", true},
		{"v0.1.2", "v0.1.2", false},
		{"v0.1.1", "v0.1.2", false},
		{"v0.1.3", "dev", false},
		{"pas-un-tag", "v0.1.2", false},
		{"v0.1.3", "pas-un-tag", false},
	}
	for _, c := range cases {
		if got := isNewer(c.latest, c.current); got != c.want {
			t.Errorf("isNewer(%q,%q) = %v ; veut %v", c.latest, c.current, got, c.want)
		}
	}
}
```

- [ ] **Step 2 : Lancer les tests pour vérifier qu'ils échouent**

Run: `go test ./internal/app/ -run 'TestParseSemver|TestIsNewer' -v`
Expected: FAIL — `undefined: parseSemver`, `undefined: isNewer`.

- [ ] **Step 3 : Implémenter le minimum**

Créer `internal/app/update.go` :

```go
package app

import (
	"strconv"
	"strings"
)

// version est injectée au build via ldflags
// (-X github.com/damienp199/bagent/internal/app.version=vX.Y.Z).
// "dev" pour un build local : désactive la notif de mise à jour.
var version = "dev"

// parseSemver découpe "vMAJOR.MINOR.PATCH" ; ok=false si non parsable.
func parseSemver(tag string) ([3]int, bool) {
	parts := strings.Split(strings.TrimPrefix(tag, "v"), ".")
	if len(parts) != 3 {
		return [3]int{}, false
	}
	var out [3]int
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return [3]int{}, false
		}
		out[i] = n
	}
	return out, true
}

// isNewer indique si latest est strictement plus récent que current.
// Renvoie false pour un build "dev" ou un tag non parsable (jamais de fausse notif).
func isNewer(latest, current string) bool {
	if current == "dev" {
		return false
	}
	l, ok1 := parseSemver(latest)
	c, ok2 := parseSemver(current)
	if !ok1 || !ok2 {
		return false
	}
	for i := 0; i < 3; i++ {
		if l[i] != c[i] {
			return l[i] > c[i]
		}
	}
	return false
}
```

- [ ] **Step 4 : Lancer les tests pour vérifier qu'ils passent**

Run: `go test ./internal/app/ -run 'TestParseSemver|TestIsNewer' -v`
Expected: PASS.

- [ ] **Step 5 : Commit**

```bash
git add internal/app/update.go internal/app/update_test.go
git commit -m "Ajoute version embarquée et comparaison semver (isNewer)"
```

---

### Task 2 : Récupération de la release, cache et orchestration

**Files:**
- Modify: `internal/app/update.go`
- Test: `internal/app/update_test.go`

**Interfaces:**
- Consumes: `isNewer`, `version` (Task 1), `configDir()` (existant, `workspace.go`).
- Produces:
  - `latestRelease(fetch func(string) ([]byte, error)) (string, error)` — extrait `tag_name` de l'API.
  - `httpFetch(url string) ([]byte, error)` — fetch HTTP réel (timeout 3 s).
  - `checkForUpdate(now int64, fetch func(string) ([]byte, error)) string` — renvoie le tag à proposer (`""` si rien) ; gère cache 24 h ; avale toutes les erreurs.

- [ ] **Step 1 : Écrire les tests qui échouent**

Ajouter à `internal/app/update_test.go` :

```go
import (
	"errors"
	"os"
	"testing"
)

func TestLatestRelease(t *testing.T) {
	fetch := func(string) ([]byte, error) {
		return []byte(`{"tag_name":"v0.1.5","name":"v0.1.5"}`), nil
	}
	tag, err := latestRelease(fetch)
	if err != nil || tag != "v0.1.5" {
		t.Fatalf("latestRelease = %q,%v ; veut v0.1.5,nil", tag, err)
	}

	boom := func(string) ([]byte, error) { return nil, errors.New("réseau") }
	if _, err := latestRelease(boom); err == nil {
		t.Fatal("latestRelease devrait propager l'erreur de fetch")
	}

	empty := func(string) ([]byte, error) { return []byte(`{}`), nil }
	if _, err := latestRelease(empty); err == nil {
		t.Fatal("latestRelease devrait échouer sur tag_name vide")
	}
}

func TestCheckForUpdate(t *testing.T) {
	t.Setenv("HOME", t.TempDir()) // isole le cache (~/.config/bagent)
	version = "v0.1.2"
	t.Cleanup(func() { version = "dev" })

	fetch := func(string) ([]byte, error) {
		return []byte(`{"tag_name":"v0.1.5"}`), nil
	}

	// 1er appel : pas de cache → fetch, notif attendue.
	if got := checkForUpdate(1000, fetch); got != "v0.1.5" {
		t.Fatalf("checkForUpdate (frais) = %q ; veut v0.1.5", got)
	}

	// 2e appel < 24h plus tard : sert le cache même si le fetch planterait.
	boom := func(string) ([]byte, error) { return nil, errors.New("réseau") }
	if got := checkForUpdate(2000, boom); got != "v0.1.5" {
		t.Fatalf("checkForUpdate (caché) = %q ; veut v0.1.5 (cache)", got)
	}

	// > 24h plus tard : cache périmé → re-fetch.
	older := func(string) ([]byte, error) {
		return []byte(`{"tag_name":"v0.1.1"}`), nil
	}
	if got := checkForUpdate(1000+86401, older); got != "" {
		t.Fatalf("checkForUpdate (périmé, v0.1.1 <= v0.1.2) = %q ; veut \"\"", got)
	}

	// Erreur réseau sans cache valide → silencieux.
	os.RemoveAll(updateCacheFile())
	if got := checkForUpdate(99999999, boom); got != "" {
		t.Fatalf("checkForUpdate (erreur) = %q ; veut \"\"", got)
	}
}
```

- [ ] **Step 2 : Lancer les tests pour vérifier qu'ils échouent**

Run: `go test ./internal/app/ -run 'TestLatestRelease|TestCheckForUpdate' -v`
Expected: FAIL — `undefined: latestRelease`, `undefined: checkForUpdate`, `undefined: updateCacheFile`.

- [ ] **Step 3 : Implémenter le minimum**

Ajouter à `internal/app/update.go` (compléter les imports) :

```go
import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	releaseAPI     = "https://api.github.com/repos/damienp199/bagent/releases/latest"
	updateCheckTTL = 24 * 60 * 60 // secondes
)

func updateCacheFile() string { return filepath.Join(configDir(), ".update-check") }

type updateCache struct {
	CheckedAt int64  `json:"checked_at"`
	Tag       string `json:"tag"`
}

// httpFetch récupère une URL avec un timeout court. Échoue sur statut non-2xx.
func httpFetch(url string) ([]byte, error) {
	cl := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := cl.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("statut HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// latestRelease extrait le tag_name de la dernière release.
func latestRelease(fetch func(string) ([]byte, error)) (string, error) {
	b, err := fetch(releaseAPI)
	if err != nil {
		return "", err
	}
	var r struct {
		TagName string `json:"tag_name"`
	}
	if err := json.Unmarshal(b, &r); err != nil {
		return "", err
	}
	if r.TagName == "" {
		return "", errors.New("tag_name vide")
	}
	return r.TagName, nil
}

func readUpdateCache() (updateCache, bool) {
	b, err := os.ReadFile(updateCacheFile())
	if err != nil {
		return updateCache{}, false
	}
	var c updateCache
	if err := json.Unmarshal(b, &c); err != nil {
		return updateCache{}, false
	}
	return c, true
}

func writeUpdateCache(c updateCache) {
	b, err := json.Marshal(c)
	if err != nil {
		return
	}
	_ = os.MkdirAll(configDir(), 0o755)
	_ = os.WriteFile(updateCacheFile(), b, 0o644)
}

// checkForUpdate renvoie le tag à proposer ("" si rien). Sert le cache s'il a
// moins de 24h, sinon interroge l'API et le réécrit. Toute erreur → "".
func checkForUpdate(now int64, fetch func(string) ([]byte, error)) string {
	tag := ""
	if c, ok := readUpdateCache(); ok && now-c.CheckedAt < updateCheckTTL {
		tag = c.Tag
	} else {
		t, err := latestRelease(fetch)
		if err != nil {
			return ""
		}
		tag = t
		writeUpdateCache(updateCache{CheckedAt: now, Tag: tag})
	}
	if isNewer(tag, version) {
		return tag
	}
	return ""
}
```

- [ ] **Step 4 : Lancer les tests pour vérifier qu'ils passent**

Run: `go test ./internal/app/ -run 'TestLatestRelease|TestCheckForUpdate' -v`
Expected: PASS.

- [ ] **Step 5 : Commit**

```bash
git add internal/app/update.go internal/app/update_test.go
git commit -m "Ajoute la récupération de release GitHub avec cache 24h"
```

---

### Task 3 : Notif dans le TUI (footer)

**Files:**
- Modify: `internal/app/tui.go` (struct `model`, `Init`, `Update`)
- Modify: `internal/app/view.go` (`footer`)
- Test: `internal/app/update_test.go`

**Interfaces:**
- Consumes: `checkForUpdate`, `httpFetch` (Task 2).
- Produces:
  - champ `model.updateTag string`.
  - type `updateAvailableMsg{ tag string }`.
  - `updateCheckCmd() tea.Cmd`.

- [ ] **Step 1 : Écrire le test qui échoue**

Ajouter à `internal/app/update_test.go` :

```go
import "strings" // si pas déjà importé

func TestFooterShowsUpdateNotice(t *testing.T) {
	m := model{
		pages:    []Page{{Title: "Favoris", Icon: "★", Kind: KindFavoris}},
		mode:     modeList,
		focus:    focusList,
		width:    100,
		height:   30,
		updateTag: "v0.1.9",
	}
	out := stripANSI(m.footer())
	if !strings.Contains(out, "v0.1.9") || !strings.Contains(out, "bagent --update") {
		t.Fatalf("le footer devrait annoncer la mise à jour :\n%s", out)
	}

	// Un status ponctuel a priorité et masque la notif.
	m.status = "✓ Fait"
	out = stripANSI(m.footer())
	if strings.Contains(out, "bagent --update") {
		t.Fatalf("le status doit masquer la notif :\n%s", out)
	}
}
```

- [ ] **Step 2 : Lancer le test pour vérifier qu'il échoue**

Run: `go test ./internal/app/ -run TestFooterShowsUpdateNotice -v`
Expected: FAIL — `unknown field 'updateTag' in struct literal`.

- [ ] **Step 3 : Implémenter**

Dans `internal/app/tui.go`, ajouter le champ à la struct `model` (après `target string`) :

```go
	action string // résultat à exécuter après la sortie
	target string

	updateTag string // tag d'une mise à jour disponible ("" sinon)
```

Toujours dans `tui.go`, remplacer `Init` :

```go
func (m model) Init() tea.Cmd { return tea.Batch(refreshCmd(), updateCheckCmd()) }
```

Et ajouter, près de `refreshCmd` :

```go
type updateAvailableMsg struct{ tag string }

func updateCheckCmd() tea.Cmd {
	return func() tea.Msg {
		return updateAvailableMsg{tag: checkForUpdate(time.Now().Unix(), httpFetch)}
	}
}
```

Dans `Update`, ajouter un case (à côté de `refreshMsg`) :

```go
	case updateAvailableMsg:
		m.updateTag = msg.tag
		return m, nil
```

Dans `internal/app/view.go`, modifier la branche `default` de `footer()` :

```go
	default:
		if m.status != "" {
			return "\n  " + m.status
		}
		keys := m.footerKeys()
		if m.updateTag != "" {
			keys += stFooter.Render("   ") + stFav.Render("●") +
				stFooter.Render(" "+m.updateTag+" dispo · ") + stKey.Render("bagent --update")
		}
		return "\n  " + keys
	}
```

- [ ] **Step 4 : Lancer les tests pour vérifier qu'ils passent**

Run: `go test ./internal/app/ -run TestFooterShowsUpdateNotice -v && go build ./...`
Expected: PASS, build OK.

- [ ] **Step 5 : Commit**

```bash
git add internal/app/tui.go internal/app/view.go internal/app/update_test.go
git commit -m "Affiche une notif de mise à jour dans le footer du TUI"
```

---

### Task 4 : Commandes CLI `--update` et `--version`

**Files:**
- Modify: `internal/app/app.go` (`Run`, `printHelp`, nouveau `runUpdate`)
- Modify: `internal/app/update.go` (rien — `version` déjà dispo)
- Test: `internal/app/update_test.go`

**Interfaces:**
- Consumes: `version` (Task 1).
- Produces: `runUpdate()` ; nouveaux cas CLI `--update`/`-u`, `--version`/`-v`.

- [ ] **Step 1 : Écrire le test qui échoue**

Ajouter à `internal/app/update_test.go` un test sur la commande shell construite (sans l'exécuter) :

```go
func TestUpdateCommandString(t *testing.T) {
	got := updateShellCmd()
	if !strings.Contains(got, "install.sh") || !strings.Contains(got, "curl -fsSL") {
		t.Fatalf("updateShellCmd = %q ; doit lancer install.sh via curl", got)
	}
}
```

- [ ] **Step 2 : Lancer le test pour vérifier qu'il échoue**

Run: `go test ./internal/app/ -run TestUpdateCommandString -v`
Expected: FAIL — `undefined: updateShellCmd`.

- [ ] **Step 3 : Implémenter**

Dans `internal/app/update.go`, ajouter :

```go
// updateShellCmd renvoie la commande shell qui (ré)installe bagent depuis la
// dernière release, isolée pour être testable sans exécution.
func updateShellCmd() string {
	return "curl -fsSL https://raw.githubusercontent.com/damienp199/bagent/main/install.sh | sh"
}
```

Dans `internal/app/app.go`, compléter les imports avec `os/exec`, puis ajouter les cas dans le `switch os.Args[1]` de `Run` :

```go
		case "--update", "-u":
			runUpdate()
			return
		case "--version", "-v":
			fmt.Println("  bagent", version)
			return
```

Toujours dans `app.go`, ajouter la fonction :

```go
// runUpdate réinstalle bagent via install.sh (download + re-signature + mv
// atomique). La sortie est transmise au terminal.
func runUpdate() {
	c := exec.Command("sh", "-c", updateShellCmd())
	c.Stdout, c.Stderr, c.Stdin = os.Stdout, os.Stderr, os.Stdin
	if err := c.Run(); err != nil {
		os.Exit(1)
	}
}
```

Mettre à jour `printHelp()` : ajouter sous la ligne `bagent --help` :

```
    bagent --update Mettre à jour vers la dernière version
    bagent --version Afficher la version
```

- [ ] **Step 4 : Lancer les tests pour vérifier qu'ils passent**

Run: `go test ./internal/app/ -run TestUpdateCommandString -v && go vet ./... && go build ./...`
Expected: PASS, vet OK, build OK.

- [ ] **Step 5 : Commit**

```bash
git add internal/app/app.go internal/app/update.go internal/app/update_test.go
git commit -m "Ajoute les commandes bagent --update et --version"
```

---

### Task 5 : Câblage du build (ldflags) et vérification de bout en bout

**Files:**
- Aucun fichier source. Vérification manuelle + mise à jour de la procédure de release.

**Interfaces:**
- Consumes: `version` (Task 1), commandes CLI (Task 4).

- [ ] **Step 1 : Vérifier l'injection de version**

Run:
```bash
go build -ldflags "-X github.com/damienp199/bagent/internal/app.version=v0.1.3" -o /tmp/bagent-vtest . && /tmp/bagent-vtest --version
```
Expected: affiche `bagent v0.1.3`.

- [ ] **Step 2 : Vérifier le défaut `dev`**

Run: `go build -o /tmp/bagent-dev . && /tmp/bagent-dev --version`
Expected: affiche `bagent dev`.

- [ ] **Step 3 : Lancer toute la suite de tests**

Run: `go test ./...`
Expected: tout PASS.

- [ ] **Step 4 : Mettre à jour la procédure de release**

La compilation des deux cibles macOS doit désormais inclure le ldflags avec le tag publié, par exemple :

```sh
LD="-X github.com/damienp199/bagent/internal/app.version=v0.1.3"
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags "$LD" -o dist/bagent-darwin-arm64 .
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "$LD" -o dist/bagent-darwin-amd64 .
```

Mettre à jour la mémoire `bagent-release-binaire` pour refléter ce ldflags.

- [ ] **Step 5 : Commit (si des fichiers de doc/procédure ont changé)**

```bash
git add -A
git commit -m "Documente l'injection de version au build pour les releases"
```

---

## Notes d'exécution

- **Imports de `update_test.go`** : les tâches ajoutent des tests au fil de l'eau. Maintenir **un seul** bloc `import` en tête du fichier et y fusionner les paquets (`testing`, `errors`, `os`, `strings`) sans doublon — ne pas recopier littéralement les `import` de chaque tâche. `stripANSI` et `press` sont déjà définis dans `nav_test.go` (même package) : ne pas les redéfinir.
- Les tests manipulant le cache isolent `HOME` via `t.Setenv("HOME", t.TempDir())` et restaurent `version` via `t.Cleanup`.
- La notif n'apparaît qu'en `modeList` (le footer des autres modes est inchangé), ce qui est le comportement voulu.
- Après livraison, la première release publiée avec le ldflags fera apparaître la notif chez les clients exécutant une version antérieure.
