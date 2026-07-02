package app

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAlfredItemFormat(t *testing.T) {
	pages := []Page{
		{Title: "Dev", Kind: KindProjet, Items: []Item{
			{Name: "bagent", FullPath: "/Users/x/Documents/Dev/bagent"},
		}},
	}
	res := alfredResultFrom(pages, map[string]bool{})
	if len(res.Items) != 1 {
		t.Fatalf("attendu 1 item, obtenu %d", len(res.Items))
	}
	it := res.Items[0]
	if it.Title != "[DEV] bagent" {
		t.Errorf("title = %q, attendu %q", it.Title, "[DEV] bagent")
	}
	if it.Arg != "/Users/x/Documents/Dev/bagent" {
		t.Errorf("arg = %q", it.Arg)
	}
	if it.UID != it.Arg {
		t.Errorf("uid = %q, attendu = arg", it.UID)
	}
	if it.Subtitle != "↵ VSCode" {
		t.Errorf("subtitle = %q", it.Subtitle)
	}
	if it.Match != "bagent dev" {
		t.Errorf("match = %q, attendu %q", it.Match, "bagent dev")
	}
	if it.Icon == nil || it.Icon.Type != "fileicon" || it.Icon.Path != it.Arg {
		t.Errorf("icon = %+v", it.Icon)
	}
	if it.Mods["cmd"].Subtitle != "Ouvrir dans Claude Code" {
		t.Errorf("mod cmd = %q", it.Mods["cmd"].Subtitle)
	}
	if it.Mods["alt"].Subtitle != "Ouvrir dans Codex" {
		t.Errorf("mod alt = %q", it.Mods["alt"].Subtitle)
	}
}

func TestAlfredFavoriteStar(t *testing.T) {
	path := "/Users/x/Documents/Dev/bagent"
	pages := []Page{
		{Kind: KindProjet, Items: []Item{{Name: "bagent", FullPath: path}}},
	}
	res := alfredResultFrom(pages, map[string]bool{path: true})
	if got := res.Items[0].Title; got != "[DEV] bagent ★" {
		t.Errorf("title favori = %q, attendu %q", got, "[DEV] bagent ★")
	}
}

func TestAlfredDedupByPath(t *testing.T) {
	path := "/Users/x/Documents/Dev/bagent"
	// Même chemin dans la page Favoris et dans un projet.
	pages := []Page{
		{Kind: KindFavoris, Items: []Item{{Name: "bagent", FullPath: path, Fav: true}}},
		{Kind: KindProjet, Items: []Item{{Name: "bagent", FullPath: path}}},
	}
	res := alfredResultFrom(pages, map[string]bool{path: true})
	if len(res.Items) != 1 {
		t.Fatalf("attendu 1 item après dédup, obtenu %d", len(res.Items))
	}
	// L'entrée unique porte son vrai groupe [DEV] et le ★.
	if got := res.Items[0].Title; got != "[DEV] bagent ★" {
		t.Errorf("title = %q, attendu %q", got, "[DEV] bagent ★")
	}
}

func TestAlfredEmptyState(t *testing.T) {
	res := alfredResultFrom(nil, map[string]bool{})
	if len(res.Items) != 1 {
		t.Fatalf("attendu 1 item d'état vide, obtenu %d", len(res.Items))
	}
	it := res.Items[0]
	if it.Valid == nil || *it.Valid {
		t.Errorf("état vide doit être valid:false, obtenu %+v", it.Valid)
	}
	if it.Arg != "" {
		t.Errorf("état vide ne doit pas avoir d'arg, obtenu %q", it.Arg)
	}
}

// Le JSON produit ne doit pas contenir de champ valid pour un item normal
// (défaut Alfred = true) ni de subtitle vide superflu.
func TestAlfredJSONOmitsValidOnNormalItem(t *testing.T) {
	pages := []Page{{Kind: KindProjet, Items: []Item{
		{Name: "bagent", FullPath: "/Users/x/Documents/Dev/bagent"},
	}}}
	res := alfredResultFrom(pages, map[string]bool{})
	b, err := json.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), `"valid"`) {
		t.Errorf("le JSON ne doit pas contenir \"valid\" pour un item normal : %s", b)
	}
}
