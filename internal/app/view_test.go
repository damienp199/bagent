package app

import (
	"strings"
	"testing"
)

// La vue doit remplir exactement la hauteur du terminal (frame plein écran).
func TestViewFillsHeight(t *testing.T) {
	for _, h := range []int{20, 30, 50} {
		m := newModel()
		m.width = 100
		m.height = h
		got := strings.Count(m.View(), "\n") + 1
		if got != h {
			t.Errorf("height=%d : View a %d lignes, attendu %d", h, got, h)
		}
	}
}

// buildPages : Récents en tête, Favoris présent ensuite.
func TestBuildPages(t *testing.T) {
	pages := buildPages()
	if len(pages) < 2 {
		t.Fatalf("attendu >=2 pages, obtenu %d", len(pages))
	}
	if pages[0].Kind != KindRecents {
		t.Errorf("1re page doit être Récents, obtenu %v", pages[0].Title)
	}
	if pages[favorisIndex(pages)].Kind != KindFavoris {
		t.Errorf("page Favoris introuvable")
	}
}
