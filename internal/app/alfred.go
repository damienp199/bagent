package app

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

// Sortie « Script Filter » d'Alfred (JSON sur stdout).
// https://www.alfredapp.com/help/workflows/inputs/script-filter/json/
//
// Le filtrage est fait ici, pas par Alfred (« Alfred Filters Results » est
// décoché dans le workflow) : le matching natif d'Alfred ne matche qu'en début
// de mot, alors qu'on veut du sous-chaîne (« reach » → « pme-outreach-podcast »).

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
		Icon:     &alfredIcon{Type: "fileicon", Path: path},
		Mods: map[string]alfredMod{
			"cmd": {Subtitle: "Ouvrir dans Claude Code"},
			"alt": {Subtitle: "Ouvrir dans Codex"},
		},
	}
}

// alfredMatch teste si tous les termes de la requête sont des sous-chaînes du
// nom ou du groupe (insensible à la casse). Un item passe si chaque mot tapé
// apparaît quelque part — « reach » matche « outreach », « pme podcast » aussi.
func alfredMatch(path, query string) bool {
	hay := strings.ToLower(filepath.Base(path) + " " + filepath.Base(filepath.Dir(path)))
	for _, term := range strings.Fields(strings.ToLower(query)) {
		if !strings.Contains(hay, term) {
			return false
		}
	}
	return true
}

// alfredResultFrom aplatit les pages en une liste unique, dédupliquée par
// chemin complet, puis ne garde que les items correspondant à la requête. Un
// dossier présent dans les favoris et sous un projet n'apparaît qu'une fois.
func alfredResultFrom(pages []Page, favs map[string]bool, query string) alfredResult {
	seen := map[string]bool{}
	var items []alfredItem
	for _, p := range pages {
		for _, it := range p.Items {
			if seen[it.FullPath] {
				continue
			}
			seen[it.FullPath] = true
			if !alfredMatch(it.FullPath, query) {
				continue
			}
			items = append(items, alfredItemFor(it.FullPath, favs[it.FullPath]))
		}
	}
	if len(items) == 0 {
		no := false
		title, sub := "Aucun résultat", "Aucun workspace ne correspond à « "+strings.TrimSpace(query)+" »"
		if strings.TrimSpace(query) == "" {
			title, sub = "Aucun workspace configuré", "Configure via bagent"
		}
		items = []alfredItem{{Title: title, Subtitle: sub, Valid: &no}}
	}
	return alfredResult{Items: items}
}

// runAlfred imprime le JSON Script Filter sur stdout, lu en direct depuis la
// config à chaque frappe dans Alfred. La requête tapée arrive en argument.
func runAlfred(args []string) {
	query := strings.Join(args, " ")
	res := alfredResultFrom(buildPages(), favoriteSet(), query)
	b, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		fmt.Println(`{"items":[]}`)
		return
	}
	fmt.Println(string(b))
}
