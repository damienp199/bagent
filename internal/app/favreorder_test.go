package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// swapFavorites échange deux favoris dans le fichier favorites (sans préfixe).
func TestSwapFavoritesFile(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if err := writeLines(favoritesFile(), []string{"/p/a", "/p/dead", "/p/c"}); err != nil {
		t.Fatalf("writeLines: %v", err)
	}
	if !swapFavorites("/p/a", "/p/c") {
		t.Fatal("swapFavorites(/p/a, /p/c) devrait échanger")
	}
	got := strings.Join(loadFavorites(), ",")
	if want := "/p/c,/p/dead,/p/a"; got != want {
		t.Fatalf("après échange, ordre = %q, attendu %q", got, want)
	}
	if swapFavorites("/p/x", "/p/c") {
		t.Fatal("entrée absente : no-op attendu")
	}
}

// favReorderModel construit un model avec trois favoris réels, focus sur la
// liste, le premier favori sélectionné.
func favReorderModel(t *testing.T) (model, string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	d := filepath.Join(home, "Documents")
	for _, n := range []string{"A", "B", "C"} {
		if err := os.MkdirAll(filepath.Join(d, n), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}
	writeLines(favoritesFile(), []string{
		filepath.Join(d, "A"),
		filepath.Join(d, "B"),
		filepath.Join(d, "C"),
	})
	m := newModel()
	m.pageIdx = favorisIndex(m.pages)
	m.focus = focusList
	m.selected = 0
	return m, d
}

// 'o' depuis la liste des favoris active le mode ordre ; esc en sort.
func TestReorderFavModeToggle(t *testing.T) {
	m, _ := favReorderModel(t)
	m = press(m, "o")
	if m.mode != modeReorder {
		t.Fatalf("après 'o' sur la liste Favoris, mode = %v, attendu modeReorder", m.mode)
	}
	m = press(m, "esc")
	if m.mode != modeList {
		t.Fatalf("après 'esc', mode = %v, attendu modeList", m.mode)
	}
}

// En mode ordre, ↓ descend le favori sélectionné et persiste l'ordre.
func TestMoveFavoriteDown(t *testing.T) {
	m, d := favReorderModel(t)
	m = press(m, "o")
	m = press(m, "down") // A descend d'un cran : B, A, C
	got := strings.Join(loadFavorites(), ",")
	want := strings.Join([]string{
		filepath.Join(d, "B"),
		filepath.Join(d, "A"),
		filepath.Join(d, "C"),
	}, ",")
	if got != want {
		t.Fatalf("après ↓ : ordre = %q, attendu %q", got, want)
	}
	// La sélection suit le favori déplacé (A est désormais en position 1).
	if it, ok := m.current(); !ok || filepath.Base(it.FullPath) != "A" {
		t.Fatalf("la sélection devrait suivre A, obtenu %+v (ok=%v)", it, ok)
	}
}

// En mode ordre favoris, le favori sélectionné affiche l'indicateur ↕ et le
// footer propose ↑/↓.
func TestReorderFavViewIndicators(t *testing.T) {
	m, _ := favReorderModel(t)
	m = press(m, "o")
	out := stripANSI(m.View())
	if !strings.Contains(out, "↕") {
		t.Fatalf("le favori en cours d'ordre devrait afficher ↕ :\n%s", out)
	}
	if !strings.Contains(out, "↑/↓") {
		t.Fatalf("le footer devrait proposer ↑/↓ :\n%s", out)
	}
}

// En mode ordre, ↑ sur le premier favori est un no-op (bord haut).
func TestMoveFavoriteUpAtTopIsNoop(t *testing.T) {
	m, d := favReorderModel(t)
	m = press(m, "o")
	m = press(m, "up")
	got := strings.Join(loadFavorites(), ",")
	want := strings.Join([]string{
		filepath.Join(d, "A"),
		filepath.Join(d, "B"),
		filepath.Join(d, "C"),
	}, ",")
	if got != want {
		t.Fatalf("↑ au bord haut devrait être un no-op : ordre = %q, attendu %q", got, want)
	}
}
