package app

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

// Sortie « Script Filter » d'Alfred (JSON sur stdout).
// https://www.alfredapp.com/help/workflows/inputs/script-filter/json/

type alfredIcon struct {
	Type string `json:"type"`
	Path string `json:"path"`
}

type alfredMod struct {
	Subtitle string `json:"subtitle"`
}

type alfredItem struct {
	UID      string               `json:"uid,omitempty"`
	Title    string               `json:"title"`
	Subtitle string               `json:"subtitle,omitempty"`
	Arg      string               `json:"arg,omitempty"`
	Match    string               `json:"match,omitempty"`
	Valid    *bool                `json:"valid,omitempty"`
	Icon     *alfredIcon          `json:"icon,omitempty"`
	Mods     map[string]alfredMod `json:"mods,omitempty"`
}

type alfredResult struct {
	Items []alfredItem `json:"items"`
}

// alfredItemFor construit l'item Alfred d'un workspace. Le groupe et le ★
// dérivent du chemin (pas de la page source), pour être stables après dédup.
func alfredItemFor(path string, fav bool) alfredItem {
	name := filepath.Base(path)
	group := strings.ToUpper(filepath.Base(filepath.Dir(path)))
	title := "[" + group + "] " + name
	if fav {
		title += " ★"
	}
	return alfredItem{
		UID:      path,
		Title:    title,
		Subtitle: "↵ VSCode",
		Arg:      path,
		Match:    strings.ToLower(name + " " + group),
		Icon:     &alfredIcon{Type: "fileicon", Path: path},
		Mods: map[string]alfredMod{
			"cmd": {Subtitle: "Ouvrir dans Claude Code"},
			"alt": {Subtitle: "Ouvrir dans Codex"},
		},
	}
}

// alfredResultFrom aplatit les pages en une liste unique, dédupliquée par
// chemin complet. Un dossier présent dans les favoris et sous un projet
// n'apparaît qu'une fois.
func alfredResultFrom(pages []Page, favs map[string]bool) alfredResult {
	seen := map[string]bool{}
	var items []alfredItem
	for _, p := range pages {
		for _, it := range p.Items {
			if seen[it.FullPath] {
				continue
			}
			seen[it.FullPath] = true
			items = append(items, alfredItemFor(it.FullPath, favs[it.FullPath]))
		}
	}
	if len(items) == 0 {
		no := false
		items = []alfredItem{{
			Title:    "Aucun workspace configuré",
			Subtitle: "Configure via bagent",
			Valid:    &no,
		}}
	}
	return alfredResult{Items: items}
}

// runAlfred imprime le JSON Script Filter sur stdout, lu en direct depuis la
// config à chaque frappe dans Alfred.
func runAlfred() {
	res := alfredResultFrom(buildPages(), favoriteSet())
	b, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		fmt.Println(`{"items":[]}`)
		return
	}
	fmt.Println(string(b))
}
