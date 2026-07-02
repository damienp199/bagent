package app

import "testing"

func TestWorkspacesFrom(t *testing.T) {
	pages := []Page{
		{Kind: KindProjet, Items: []Item{
			{Name: "bagent", FullPath: "/Users/x/Documents/Dev/bagent"},
		}},
	}
	out := workspacesFrom(pages, map[string]bool{"/Users/x/Documents/Dev/bagent": true})
	if len(out) != 1 {
		t.Fatalf("attendu 1 workspace, obtenu %d", len(out))
	}
	w := out[0]
	if w.Name != "bagent" {
		t.Errorf("name = %q", w.Name)
	}
	if w.Path != "/Users/x/Documents/Dev/bagent" {
		t.Errorf("path = %q", w.Path)
	}
	if w.Group != "DEV" {
		t.Errorf("group = %q, attendu DEV", w.Group)
	}
	if !w.Fav {
		t.Errorf("fav = false, attendu true")
	}
}

func TestWorkspacesFromDedup(t *testing.T) {
	path := "/Users/x/Documents/Dev/bagent"
	pages := []Page{
		{Kind: KindFavoris, Items: []Item{{Name: "bagent", FullPath: path, Fav: true}}},
		{Kind: KindProjet, Items: []Item{{Name: "bagent", FullPath: path}}},
	}
	out := workspacesFrom(pages, map[string]bool{path: true})
	if len(out) != 1 {
		t.Fatalf("attendu 1 après dédup, obtenu %d", len(out))
	}
	if out[0].Group != "DEV" || !out[0].Fav {
		t.Errorf("entrée dédupliquée incorrecte: %+v", out[0])
	}
}
