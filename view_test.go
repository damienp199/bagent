package main

import (
	"strings"
	"testing"
)

// La vue doit remplir exactement la hauteur du terminal (frame plein écran),
// sinon le contenu peut se décaler dans l'écran alternatif.
func TestViewFillsHeight(t *testing.T) {
	for _, h := range []int{20, 30, 50} {
		m := newModel()
		m.width = 100
		m.height = h
		m.recomputeScroll()
		got := strings.Count(m.View(), "\n") + 1
		if got != h {
			t.Errorf("height=%d : View a %d lignes, attendu %d", h, got, h)
		}
	}
}
