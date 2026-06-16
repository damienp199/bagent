package app

import (
	"os"
	"path/filepath"
	"testing"
)

// Dans l'onglet Favoris, 'f' retire l'item courant des favoris (sans confirmation).
func TestFavorisRemoveWithF(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	d := filepath.Join(home, "Documents")
	fav := filepath.Join(d, "FAV")
	if err := os.MkdirAll(fav, 0o755); err != nil {
		t.Fatal(err)
	}
	writeLines(favoritesFile(), []string{fav})

	m := newModel()
	m.focus = focusList
	if m.curPage().Kind != KindFavoris {
		t.Fatalf("la page d'accueil devrait être Favoris, obtenu %q", m.curPage().Title)
	}
	if len(m.curPage().Items) != 1 {
		t.Fatalf("attendu 1 favori, obtenu %d", len(m.curPage().Items))
	}

	m = press(m, "f")

	if got := loadFavorites(); len(got) != 0 {
		t.Fatalf("après 'f', favoris = %v, attendu vide", got)
	}
}
