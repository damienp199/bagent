package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

// renderItemLines produit les lignes affichables et, pour chacune, l'index de
// l'item correspondant (-1 pour les lignes non sélectionnables).
func (m model) renderItemLines() ([]string, []int) {
	var lines []string
	var owner []int
	shownProjects := false
	shownRecents := false

	for i, it := range m.items {
		switch it.Type {
		case TypeGroupHeader:
			lines = append(lines, "")
			owner = append(owner, -1)
			lines = append(lines, "  "+stSection.Render("⌂ "+strings.ToUpper(it.Name)))
			owner = append(owner, -1)
			continue
		case TypeWorkspace:
			if !shownProjects {
				lines = append(lines, "  "+stSection.Render("» PROJETS"))
				owner = append(owner, -1)
				shownProjects = true
			}
		case TypeRecent:
			if !shownRecents {
				lines = append(lines, "")
				owner = append(owner, -1)
				lines = append(lines, "  "+stSection.Render("◷ RÉCENTS"))
				owner = append(owner, -1)
				shownRecents = true
			}
		}

		prefix := "- "
		if it.Type == TypeGroupItem {
			if it.LastInGroup {
				prefix = "└ "
			} else {
				prefix = "├ "
			}
		}
		numDisp := ""
		if it.Num > 0 {
			numDisp = fmt.Sprintf("%d ", it.Num)
		}
		isDefault := i == 0 && it.Type == TypeWorkspace

		var line string
		if i == m.selected {
			line = "  " + stArrow.Render("❯") + " " + stSel.Render(prefix+numDisp+it.Name)
		} else {
			line = "    " + stNormal.Render(prefix+numDisp+it.Name)
		}
		if isDefault {
			line += "  " + stDefault.Render("●")
		}
		lines = append(lines, line)
		owner = append(owner, i)
	}
	return lines, owner
}

const reservedLines = 6 // titre, blank, indicateur haut, indicateur bas, blank, footer

func (m *model) recomputeScroll() {
	_, owner := m.renderItemLines()
	n := len(owner)
	avail := m.height - reservedLines
	if avail < 3 {
		avail = 3
	}
	selLine := 0
	for idx, o := range owner {
		if o == m.selected {
			selLine = idx
			break
		}
	}
	if n <= avail {
		m.scrollTop = 0
		return
	}
	if selLine < m.scrollTop {
		m.scrollTop = selLine
	}
	if selLine > m.scrollTop+avail-1 {
		m.scrollTop = selLine - avail + 1
	}
	if maxTop := n - avail; m.scrollTop > maxTop {
		m.scrollTop = maxTop
	}
	if m.scrollTop < 0 {
		m.scrollTop = 0
	}
}

func (m model) View() string {
	lines, _ := m.renderItemLines()
	n := len(lines)
	avail := m.height - reservedLines
	if avail < 3 {
		avail = 3
	}
	top := m.scrollTop
	if n <= avail {
		top = 0
	}
	bot := top + avail
	if bot > n {
		bot = n
	}

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString("  " + stPuce.Render("●") + " " + stTitle.Render("bagent") + "\n")

	if top > 0 {
		b.WriteString("  " + stDefault.Render("▲") + " " + stDim.Render(fmt.Sprintf("%d de plus", top)) + "\n")
	} else {
		b.WriteString("\n")
	}

	for i := top; i < bot; i++ {
		b.WriteString(lines[i] + "\n")
	}
	// Remplir la zone de contenu jusqu'à `avail` lignes pour que le frame
	// occupe toute la hauteur de l'écran alternatif (évite le décalage vers le bas).
	for pad := bot - top; pad < avail; pad++ {
		b.WriteString("\n")
	}

	if bot < n {
		b.WriteString("  " + stDefault.Render("▼") + " " + stDim.Render(fmt.Sprintf("%d de plus", n-bot)) + "\n")
	} else {
		b.WriteString("\n")
	}

	b.WriteString(m.footer())
	return b.String()
}

func (m model) footer() string {
	switch m.mode {
	case modeAddPath:
		return "\n  " + stArrow.Render("›") + " Path : " + m.input.View()
	case modeAddKind:
		return "\n  " + stArrow.Render("›") + " Direct " + stKey.Render("d") + " ou parent " + stKey.Render("p") + " ?"
	case modeDelConfirm:
		it, _ := m.current()
		name := it.Name
		if it.Type == TypeGroupItem {
			name = filepath.Base(it.Group)
		}
		return "\n  " + stArrow.Render("›") + " Supprimer " + stSel.Render(name) + " ? " + stKey.Render("(o/N)")
	default:
		if m.status != "" {
			return "\n  " + m.status
		}
		return "\n  " + m.footerKeys()
	}
}

func (m model) footerKeys() string {
	key := func(k, label string) string {
		return stKey.Render(k) + stFooter.Render(" "+label)
	}
	parts := []string{
		key("↑↓", "naviguer"),
		key("⏎", "vscode"),
		key("c", "claude"),
		key("x", "codex"),
		key("a", "ajouter"),
		key("s", "suppr"),
		key("d", "défaut"),
		key("q", "quitter"),
	}
	return strings.Join(parts, "  ")
}
