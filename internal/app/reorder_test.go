package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Régression : un projet "mort" (dossier inexistant) est invisible comme onglet
// mais présent dans le fichier. Le déplacement doit raisonner sur les onglets
// VISIBLES, sans pas fantôme à travers les projets morts.
func TestReorderSkipsDeadProjects(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	d := filepath.Join(home, "Documents")
	os.MkdirAll(filepath.Join(d, "A"), 0o755)
	os.MkdirAll(filepath.Join(d, "C"), 0o755)
	// B n'existe pas → onglet invisible
	writeLines(workspacesFile(), []string{
		">" + filepath.Join(d, "A"),
		">" + filepath.Join(d, "B"),
		">" + filepath.Join(d, "C"),
	})
	m := newModel()
	m.gotoProjet(filepath.Join(d, "C")) // 2e projet visible
	m.focus = focusBar
	idxC := m.pageIdx
	m = press(m, "o")
	m = press(m, "left") // un seul ← doit faire passer C devant A
	if m.pageIdx != idxC-1 {
		t.Errorf("après 1× ← : pageIdx=%d, attendu %d (le projet mort B ne doit pas créer de pas fantôme)", m.pageIdx, idxC-1)
	}
	if base := filepath.Base(m.curPage().Parent); base != "C" {
		t.Errorf("l'onglet courant devrait rester C, obtenu %q", base)
	}
}

// reorderModel construit un model en mémoire (sans I/O) : Récents, Favoris,
// puis deux projets, le focus en haut sur le premier projet.
func reorderModel() model {
	m := model{mode: modeList, focus: focusBar, width: 100, height: 30}
	m.pages = []Page{
		{Title: "Récents", Icon: "◷", Kind: KindRecents},
		{Title: "Favoris", Icon: "★", Kind: KindFavoris},
		{Title: "alpha", Kind: KindProjet, Parent: "/p/alpha"},
		{Title: "beta", Kind: KindProjet, Parent: "/p/beta"},
	}
	m.pageIdx = 2
	return m
}

func TestReorderModeToggle(t *testing.T) {
	m := reorderModel()
	m = press(m, "o")
	if m.mode != modeReorder {
		t.Fatalf("après 'o' en haut sur un projet, mode = %v, attendu modeReorder", m.mode)
	}
	m = press(m, "esc")
	if m.mode != modeList {
		t.Fatalf("après 'esc', mode = %v, attendu modeList", m.mode)
	}
}

func TestReorderModeIgnoredOnFavoris(t *testing.T) {
	m := reorderModel()
	m.pageIdx = 1 // Favoris
	m = press(m, "o")
	if m.mode == modeReorder {
		t.Fatal("'o' ne devrait pas activer le mode ordre sur Favoris")
	}
}

func TestReorderModeChevrons(t *testing.T) {
	m := reorderModel()
	m.mode = modeReorder
	out := stripANSI(m.View())
	if !strings.Contains(out, "‹") || !strings.Contains(out, "›") {
		t.Fatalf("le mode ordre devrait encadrer l'onglet de chevrons :\n%s", out)
	}
}

func TestSwapProjectsFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if err := writeLines(workspacesFile(), []string{">/p/a", ">/p/dead", ">/p/c"}); err != nil {
		t.Fatalf("writeLines: %v", err)
	}
	// échange deux projets séparés par une entrée morte
	if !swapProjects("/p/a", "/p/c") {
		t.Fatal("swapProjects(/p/a, /p/c) devrait échanger")
	}
	got := strings.Join(loadDirs(), ",")
	if want := ">/p/c,>/p/dead,>/p/a"; got != want {
		t.Fatalf("après échange, ordre = %q, attendu %q", got, want)
	}
	if swapProjects("/p/x", "/p/c") {
		t.Fatal("entrée absente : no-op attendu")
	}
}

func TestSwapEntries(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		x, y string
		want string
	}{
		{"adjacents", []string{">a", ">b", ">c"}, ">a", ">b", ">b,>a,>c"},
		{"distants (saute un mort)", []string{">a", ">dead", ">c"}, ">a", ">c", ">c,>dead,>a"},
		{"x absent", []string{">a", ">b"}, ">z", ">a", ">a,>b"},
		{"y absent", []string{">a", ">b"}, ">a", ">z", ">a,>b"},
		{"même entrée", []string{">a", ">b"}, ">a", ">a", ">a,>b"},
	}
	for _, c := range cases {
		got := strings.Join(swapEntries(c.in, c.x, c.y), ",")
		if got != c.want {
			t.Errorf("%s: swapEntries(%v, %q, %q) = %q, attendu %q",
				c.name, c.in, c.x, c.y, got, c.want)
		}
	}
}

func TestPruneDeadEntries(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	d := filepath.Join(home, "Documents")
	os.MkdirAll(filepath.Join(d, "A"), 0o755)
	os.MkdirAll(filepath.Join(d, "C"), 0o755)
	os.MkdirAll(filepath.Join(d, "FAV"), 0o755)
	writeLines(workspacesFile(), []string{
		">" + filepath.Join(d, "A"),
		">" + filepath.Join(d, "B"), // dossier inexistant
		">" + filepath.Join(d, "C"),
	})
	writeLines(favoritesFile(), []string{
		filepath.Join(d, "FAV"),
		filepath.Join(d, "GONE"), // favori mort
	})
	if !pruneDeadEntries() {
		t.Fatal("pruneDeadEntries devrait nettoyer")
	}
	if got, want := strings.Join(loadDirs(), ","), ">"+filepath.Join(d, "A")+",>"+filepath.Join(d, "C"); got != want {
		t.Fatalf("workspaces après purge = %q, attendu %q", got, want)
	}
	if got, want := strings.Join(loadFavorites(), ","), filepath.Join(d, "FAV"); got != want {
		t.Fatalf("favoris après purge = %q, attendu %q", got, want)
	}
	if pruneDeadEntries() {
		t.Fatal("rien à nettoyer : no-op attendu")
	}
}
