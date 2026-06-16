package app

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// PageKind distingue les natures de pages.
type PageKind int

const (
	KindFavoris PageKind = iota
	KindProjet
	KindRecents
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

const maxRecents = 8

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

// resolveClaudePath convertit un nom de dossier encodé par Claude en chemin réel.
func resolveClaudePath(encoded string) string {
	encoded = strings.TrimPrefix(encoded, "-")
	parts := strings.Split(encoded, "-")
	path := ""
	i := 0
	for i < len(parts) {
		candidate := path + "/" + parts[i]
		if isDir(candidate) {
			path = candidate
			i++
			continue
		}
		found := false
		for j := i + 1; j < len(parts); j++ {
			mergedDash := parts[i]
			mergedSpace := parts[i]
			for k := i + 1; k <= j; k++ {
				mergedDash += "-" + parts[k]
				mergedSpace += " " + parts[k]
			}
			if isDir(path + "/" + mergedDash) {
				path = path + "/" + mergedDash
				i = j + 1
				found = true
				break
			} else if isDir(path + "/" + mergedSpace) {
				path = path + "/" + mergedSpace
				i = j + 1
				found = true
				break
			}
		}
		if !found {
			return ""
		}
	}
	return path
}

// loadRecents lit ~/.claude/projects par date décroissante, en excluant le home
// et les workspaces déjà épinglés (directs ou sous un groupe).
func loadRecents(dirs []string) []string {
	home, _ := os.UserHomeDir()
	projects := filepath.Join(home, ".claude", "projects")
	entries, err := os.ReadDir(projects)
	if err != nil {
		return nil
	}
	type ent struct {
		name    string
		modTime int64
	}
	var list []ent
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		list = append(list, ent{e.Name(), info.ModTime().UnixNano()})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].modTime > list[j].modTime })

	var recents []string
	for _, e := range list {
		realPath := resolveClaudePath(e.name)
		if realPath == "" || !isDir(realPath) || realPath == home {
			continue
		}
		already := false
		for _, d := range dirs {
			if d == realPath {
				already = true
				break
			}
			if strings.HasPrefix(d, ">") {
				parent := strings.TrimPrefix(d, ">")
				if strings.HasPrefix(realPath, parent+"/") {
					already = true
					break
				}
			}
		}
		if already {
			continue
		}
		recents = append(recents, realPath)
		if len(recents) >= maxRecents {
			break
		}
	}
	return recents
}

// buildPages construit les pages dans l'ordre des onglets :
// ◷ Récents · ★ Favoris · un projet par page.
func buildPages() []Page {
	dirs := loadDirs()
	favs := favoriteSet()

	// Récents (à gauche)
	var recItems []Item
	for _, r := range loadRecents(dirs) {
		recItems = append(recItems, Item{Name: filepath.Base(r), FullPath: r, Fav: favs[r]})
	}

	// Favoris
	var favItems []Item
	for _, f := range loadFavorites() {
		if !isDir(f) {
			continue
		}
		favItems = append(favItems, Item{Name: filepath.Base(f), FullPath: f, Fav: true})
	}

	pages := []Page{
		{Title: "Récents", Icon: "◷", Kind: KindRecents, Items: recItems},
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

// remapPath réécrit une ligne de config si elle référence old (exact ou préfixe),
// en conservant l'éventuel préfixe de groupe ">".
func remapPath(line, old, newp string) string {
	prefix := ""
	p := line
	if strings.HasPrefix(p, ">") {
		prefix, p = ">", p[1:]
	}
	if p == old {
		return prefix + newp
	}
	if strings.HasPrefix(p, old+"/") {
		return prefix + newp + p[len(old):]
	}
	return line
}

// updatePathRefs met à jour workspaces et favorites après un renommage de dossier.
func updatePathRefs(old, newp string) {
	for _, file := range []string{workspacesFile(), favoritesFile()} {
		lines := readLines(file)
		changed := false
		for i, l := range lines {
			if nl := remapPath(l, old, newp); nl != l {
				lines[i] = nl
				changed = true
			}
		}
		if changed {
			_ = writeLines(file, lines)
		}
	}
}

// trashDir déplace un dossier vers ~/.Trash (jamais rm), en suffixant la date
// si le nom existe déjà. Met à jour les références config.
func trashDir(path string) error {
	home, _ := os.UserHomeDir()
	trash := filepath.Join(home, ".Trash")
	if err := os.MkdirAll(trash, 0o755); err != nil {
		return err
	}
	dest := filepath.Join(trash, filepath.Base(path))
	if _, err := os.Stat(dest); err == nil {
		dest = filepath.Join(trash, filepath.Base(path)+" "+time.Now().Format("2006-01-02 150405"))
	}
	if err := os.Rename(path, dest); err != nil {
		return err
	}
	updatePathRefs(path, dest)
	return nil
}

// renameDir renomme un dossier (même parent) et met à jour les références.
func renameDir(old, newName string) (string, error) {
	parent := filepath.Dir(old)
	newPath := filepath.Join(parent, newName)
	if newPath == old {
		return old, nil
	}
	if isDir(newPath) {
		return newPath, os.ErrExist
	}
	if err := os.Rename(old, newPath); err != nil {
		return "", err
	}
	updatePathRefs(old, newPath)
	return newPath, nil
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
// tous les fichiers de chemins persistés. Les récents sont filtrés
// dynamiquement à chaque build, donc rien à purger côté persistance.
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
