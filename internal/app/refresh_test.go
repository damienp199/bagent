package app

import "testing"

func TestLocatePage(t *testing.T) {
	pages := []Page{
		{Title: "Favoris", Kind: KindFavoris},
		{Title: "A", Kind: KindProjet, Parent: "/p/a"},
		{Title: "B", Kind: KindProjet, Parent: "/p/b"},
	}
	if got := locatePage(pages, KindProjet, "/p/b"); got != 2 {
		t.Errorf("locatePage B = %d, attendu 2", got)
	}
	if got := locatePage(pages, KindFavoris, ""); got != 0 {
		t.Errorf("locatePage Favoris = %d, attendu 0", got)
	}
	if got := locatePage(pages, KindProjet, "/p/x"); got != -1 {
		t.Errorf("locatePage absent = %d, attendu -1", got)
	}
}

// reapplyPages doit suivre le projet courant (par parent) et l'item courant
// (par chemin) même si l'ordre des onglets et le contenu ont changé.
func TestReapplyPreservesSelectionAcrossReorder(t *testing.T) {
	m := model{}
	m.pages = []Page{
		{Kind: KindFavoris},
		{Kind: KindProjet, Parent: "/p/a", Items: []Item{{FullPath: "/p/a/one"}}},
		{Kind: KindProjet, Parent: "/p/b", Items: []Item{{FullPath: "/p/b/one"}, {FullPath: "/p/b/two"}}},
	}
	m.pageIdx = 2
	m.selected = 1 // /p/b/two

	// A et B inversés sur disque, + un sous-dossier ajouté à B.
	m.reapplyPages([]Page{
		{Kind: KindFavoris},
		{Kind: KindProjet, Parent: "/p/b", Items: []Item{{FullPath: "/p/b/zero"}, {FullPath: "/p/b/one"}, {FullPath: "/p/b/two"}}},
		{Kind: KindProjet, Parent: "/p/a", Items: []Item{{FullPath: "/p/a/one"}}},
	})

	if m.pageIdx != 1 {
		t.Errorf("pageIdx = %d, attendu 1 (suit le projet B)", m.pageIdx)
	}
	if it, _ := m.current(); it.FullPath != "/p/b/two" {
		t.Errorf("item = %q, attendu /p/b/two", it.FullPath)
	}
}

// Item sélectionné supprimé en externe : la sélection est clampée, pas hors borne.
func TestReapplyClampsWhenItemRemoved(t *testing.T) {
	m := model{}
	m.pages = []Page{
		{Kind: KindFavoris},
		{Kind: KindProjet, Parent: "/p/a", Items: []Item{{FullPath: "/p/a/one"}, {FullPath: "/p/a/two"}}},
	}
	m.pageIdx = 1
	m.selected = 1 // /p/a/two

	m.reapplyPages([]Page{
		{Kind: KindFavoris},
		{Kind: KindProjet, Parent: "/p/a", Items: []Item{{FullPath: "/p/a/one"}}},
	})

	if m.pageIdx != 1 {
		t.Errorf("pageIdx = %d, attendu 1", m.pageIdx)
	}
	if m.selected != 0 {
		t.Errorf("selected = %d, attendu 0 (clampé)", m.selected)
	}
}

// Projet courant supprimé en externe : repli sur l'onglet Favoris.
func TestReapplyFallsBackWhenPageGone(t *testing.T) {
	m := model{}
	m.pages = []Page{
		{Kind: KindFavoris},
		{Kind: KindProjet, Parent: "/p/a", Items: []Item{{FullPath: "/p/a/one"}}},
	}
	m.pageIdx = 1
	m.selected = 0

	m.reapplyPages([]Page{{Kind: KindFavoris}})

	if m.curPage().Kind != KindFavoris {
		t.Errorf("attendu repli sur Favoris, page courante = %v", m.curPage().Kind)
	}
}
