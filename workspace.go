package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ItemType distingue les différentes natures de lignes affichées.
type ItemType int

const (
	TypeWorkspace ItemType = iota
	TypeGroupHeader
	TypeGroupItem
	TypeRecent
)

// Item est une entrée de la liste (sélectionnable ou non pour les headers).
type Item struct {
	Name        string
	Type        ItemType
	FullPath    string
	Group       string // chemin du parent, pour les group-item
	LastInGroup bool
	Num         int // raccourci 1-9, 0 si aucun
}

const maxRecents = 5

func configDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "bagent")
}

func workspacesFile() string {
	return filepath.Join(configDir(), "workspaces")
}

func isDir(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

// loadDirs lit le fichier workspaces, une entrée par ligne (vides ignorées).
func loadDirs() []string {
	data, err := os.ReadFile(workspacesFile())
	if err != nil {
		return nil
	}
	var dirs []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		dirs = append(dirs, line)
	}
	return dirs
}

// loadSubdirs renvoie les sous-dossiers directs d'un dossier parent, triés.
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
// Claude encode les paths : / et espaces deviennent des tirets.
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

// loadRecents lit ~/.claude/projects (par date décroissante) en excluant ceux
// déjà épinglés (directs ou sous un groupe parent) et le home.
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
		if realPath == "" || !isDir(realPath) {
			continue
		}
		if realPath == home {
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

// buildItems construit la liste complète : workspaces, groupes, récents.
// Renvoie aussi l'index (1-based) du premier récent.
func buildItems() ([]Item, int) {
	dirs := loadDirs()
	recents := loadRecents(dirs)

	var items []Item
	numCounter := 0

	for _, d := range dirs {
		if strings.HasPrefix(d, ">") {
			parent := strings.TrimPrefix(d, ">")
			if !isDir(parent) {
				continue
			}
			subdirs := loadSubdirs(parent)
			if len(subdirs) == 0 {
				continue
			}
			// Exclure les sous-dossiers déjà épinglés en direct.
			var filtered []string
			for _, sd := range subdirs {
				direct := false
				for _, dd := range dirs {
					if dd == sd {
						direct = true
						break
					}
				}
				if !direct {
					filtered = append(filtered, sd)
				}
			}
			if len(filtered) == 0 {
				continue
			}
			items = append(items, Item{
				Name: filepath.Base(parent),
				Type: TypeGroupHeader,
				FullPath: parent,
			})
			for idx, sd := range filtered {
				numCounter++
				num := 0
				if numCounter <= 9 {
					num = numCounter
				}
				items = append(items, Item{
					Name:        filepath.Base(sd),
					Type:        TypeGroupItem,
					FullPath:    sd,
					Group:       parent,
					LastInGroup: idx == len(filtered)-1,
					Num:         num,
				})
			}
		} else {
			numCounter++
			num := 0
			if numCounter <= 9 {
				num = numCounter
			}
			items = append(items, Item{
				Name:     filepath.Base(d),
				Type:     TypeWorkspace,
				FullPath: d,
				Num:      num,
			})
		}
	}

	recentStart := len(items) + 1
	for _, d := range recents {
		numCounter++
		num := 0
		if numCounter <= 9 {
			num = numCounter
		}
		items = append(items, Item{
			Name:     filepath.Base(d),
			Type:     TypeRecent,
			FullPath: d,
			Num:      num,
		})
	}

	return items, recentStart
}

// --- Mutations du fichier workspaces ---

func writeLines(lines []string) error {
	if err := os.MkdirAll(configDir(), 0o755); err != nil {
		return err
	}
	content := strings.Join(lines, "\n")
	if content != "" {
		content += "\n"
	}
	return os.WriteFile(workspacesFile(), []byte(content), 0o644)
}

func rawLines() []string {
	return loadDirs()
}

func containsLine(lines []string, entry string) bool {
	for _, l := range lines {
		if l == entry {
			return true
		}
	}
	return false
}

// addEntry ajoute une entrée (déjà préfixée si groupe) si absente.
func addEntry(entry string) (added bool, err error) {
	lines := rawLines()
	if containsLine(lines, entry) {
		return false, nil
	}
	lines = append(lines, entry)
	return true, writeLines(lines)
}

// removeEntry retire une entrée exacte.
func removeEntry(entry string) error {
	lines := rawLines()
	var out []string
	for _, l := range lines {
		if l != entry {
			out = append(out, l)
		}
	}
	return writeLines(out)
}

// setDefault déplace une entrée en tête de liste.
func setDefault(entry string) error {
	lines := rawLines()
	out := []string{entry}
	for _, l := range lines {
		if l != entry {
			out = append(out, l)
		}
	}
	return writeLines(out)
}
