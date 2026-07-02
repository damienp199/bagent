package app

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

// workspaceJSON est la représentation neutre d'un workspace, consommable par
// n'importe quel lanceur (Raycast, Alfred, script perso). Émise par `bagent json`.
type workspaceJSON struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	Group string `json:"group"`
	Fav   bool   `json:"fav"`
}

// workspacesFrom aplatit les pages en une liste unique dédupliquée par chemin.
// Le groupe dérive du dossier parent du chemin (stable après dédup).
func workspacesFrom(pages []Page, favs map[string]bool) []workspaceJSON {
	seen := map[string]bool{}
	var out []workspaceJSON
	for _, p := range pages {
		for _, it := range p.Items {
			if seen[it.FullPath] {
				continue
			}
			seen[it.FullPath] = true
			out = append(out, workspaceJSON{
				Name:  filepath.Base(it.FullPath),
				Path:  it.FullPath,
				Group: strings.ToUpper(filepath.Base(filepath.Dir(it.FullPath))),
				Fav:   favs[it.FullPath],
			})
		}
	}
	return out
}

// runJSON imprime la liste des workspaces en JSON neutre sur stdout.
func runJSON() {
	out := workspacesFrom(buildPages(), favoriteSet())
	if out == nil {
		out = []workspaceJSON{}
	}
	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		fmt.Println("[]")
		return
	}
	fmt.Println(string(b))
}
