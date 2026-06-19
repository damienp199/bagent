package app

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// PageKind distingue les natures de pages.
type PageKind int

const (
	KindFavoris PageKind = iota
	KindProjet
)

// Item est un workspace ouvrable affiché dans une page.
type Item struct {
	Name     string
	FullPath string
	Fav      bool // marqué comme favori
}

// Page regroupe des items sous un onglet.
type Page struct {
	Title  string
	Icon   string
	Kind   PageKind
	Parent string // chemin du dossier parent (pages de groupe)
	Items  []Item
}

func configDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "bagent")
}

func workspacesFile() string { return filepath.Join(configDir(), "workspaces") }
func favoritesFile() string  { return filepath.Join(configDir(), "favorites") }

func isDir(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

// readLines lit un fichier en ignorant les lignes vides.
func readLines(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var out []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		out = append(out, line)
	}
	return out
}

func loadDirs() []string      { return readLines(workspacesFile()) }
func loadFavorites() []string { return readLines(favoritesFile()) }

func favoriteSet() map[string]bool {
	set := map[string]bool{}
	for _, f := range loadFavorites() {
		set[f] = true
	}
	return set
}

// loadSubdirs renvoie les sous-dossiers directs d'un parent, triés.
func loadSubdirs(parent string) []string {
	entries, err := os.ReadDir(parent)
	if err != nil {
		return nil
	}
	var subdirs []string
	for _, e := range entries {
		if e.IsDir() {
			subdirs = append(subdirs, filepath.Join(parent, e.Name()))
		}
	}
	sort.Strings(subdirs)
	return subdirs
}

// buildPages construit les pages dans l'ordre des onglets :
// ★ Favoris · un projet par page.
func buildPages() []Page {
	dirs := loadDirs()
	favs := favoriteSet()

	// Favoris
	var favItems []Item
	for _, f := range loadFavorites() {
		if !isDir(f) {
			continue
		}
		favItems = append(favItems, Item{Name: filepath.Base(f), FullPath: f, Fav: true})
	}

	pages := []Page{
		{Title: "Favoris", Icon: "★", Kind: KindFavoris, Items: favItems},
	}

	// Un projet par page (lignes préfixées ">").
	for _, d := range dirs {
		if !strings.HasPrefix(d, ">") {
			continue
		}
		parent := strings.TrimPrefix(d, ">")
		if !isDir(parent) {
			continue
		}
		subs := loadSubdirs(parent)
		var items []Item
		for _, sd := range subs {
			items = append(items, Item{Name: filepath.Base(sd), FullPath: sd, Fav: favs[sd]})
		}
		pages = append(pages, Page{Title: filepath.Base(parent), Kind: KindProjet, Parent: parent, Items: items})
	}

	return pages
}

// favorisIndex renvoie l'index de la page Favoris (page d'accueil par défaut).
func favorisIndex(pages []Page) int {
	for i, p := range pages {
		if p.Kind == KindFavoris {
			return i
		}
	}
	return 0
}

// --- Mutations ---

func writeLines(path string, lines []string) error {
	if err := os.MkdirAll(configDir(), 0o755); err != nil {
		return err
	}
	content := strings.Join(lines, "\n")
	if content != "" {
		content += "\n"
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func contains(lines []string, entry string) bool {
	for _, l := range lines {
		if l == entry {
			return true
		}
	}
	return false
}

// addEntry ajoute une entrée au fichier workspaces (déjà préfixée si groupe).
func addEntry(entry string) (bool, error) {
	lines := loadDirs()
	if contains(lines, entry) {
		return false, nil
	}
	return true, writeLines(workspacesFile(), append(lines, entry))
}

// removeEntry retire une entrée exacte du fichier workspaces.
func removeEntry(entry string) error {
	var out []string
	for _, l := range loadDirs() {
		if l != entry {
			out = append(out, l)
		}
	}
	return writeLines(workspacesFile(), out)
}

// createDir crée un sous-dossier dans parent. Renvoie son chemin complet.
func createDir(parent, name string) (string, error) {
	full := filepath.Join(parent, name)
	if isDir(full) {
		return full, os.ErrExist
	}
	if err := os.MkdirAll(full, 0o755); err != nil {
		return "", err
	}
	return full, nil
}

// swapEntries échange les positions des lignes x et y. No-op si l'une est
// absente. Fonction pure (espace des lignes du fichier).
func swapEntries(lines []string, x, y string) []string {
	ix, iy := -1, -1
	for k, l := range lines {
		switch l {
		case x:
			ix = k
		case y:
			iy = k
		}
	}
	if ix < 0 || iy < 0 || ix == iy {
		return lines
	}
	out := append([]string(nil), lines...)
	out[ix], out[iy] = out[iy], out[ix]
	return out
}

// swapProjects échange les positions des deux projets a et b dans le fichier
// workspaces. Renvoie true si l'ordre a changé.
func swapProjects(a, b string) bool {
	lines := loadDirs()
	swapped := swapEntries(lines, ">"+a, ">"+b)
	if strings.Join(swapped, "\n") == strings.Join(lines, "\n") {
		return false
	}
	return writeLines(workspacesFile(), swapped) == nil
}

// swapFavorites échange les positions des deux favoris a et b dans le fichier
// favorites. Renvoie true si l'ordre a changé.
func swapFavorites(a, b string) bool {
	lines := loadFavorites()
	swapped := swapEntries(lines, a, b)
	if strings.Join(swapped, "\n") == strings.Join(lines, "\n") {
		return false
	}
	return writeLines(favoritesFile(), swapped) == nil
}

// pruneDeadFile retire d'un fichier de chemins les lignes pointant vers un
// dossier inexistant, en conservant l'éventuel préfixe ">". true si modifié.
func pruneDeadFile(path string) bool {
	lines := readLines(path)
	kept := make([]string, 0, len(lines))
	for _, l := range lines {
		if isDir(strings.TrimPrefix(l, ">")) {
			kept = append(kept, l)
		}
	}
	if len(kept) == len(lines) {
		return false
	}
	_ = writeLines(path, kept)
	return true
}

// pruneDeadEntries purge les entrées mortes (dossier déplacé ou supprimé) de
// tous les fichiers de chemins persistés.
func pruneDeadEntries() bool {
	changed := false
	for _, f := range []string{workspacesFile(), favoritesFile()} {
		if pruneDeadFile(f) {
			changed = true
		}
	}
	return changed
}

// addFavorite ajoute un chemin aux favoris s'il est absent.
func addFavorite(path string) (bool, error) {
	favs := loadFavorites()
	if contains(favs, path) {
		return false, nil
	}
	return true, writeLines(favoritesFile(), append(favs, path))
}

// removeFavorite retire un chemin des favoris.
func removeFavorite(path string) error {
	var out []string
	for _, f := range loadFavorites() {
		if f != path {
			out = append(out, f)
		}
	}
	return writeLines(favoritesFile(), out)
}

// toggleFavorite ajoute ou retire un chemin des favoris.
func toggleFavorite(path string) error {
	favs := loadFavorites()
	if contains(favs, path) {
		var out []string
		for _, f := range favs {
			if f != path {
				out = append(out, f)
			}
		}
		return writeLines(favoritesFile(), out)
	}
	return writeLines(favoritesFile(), append(favs, path))
}
