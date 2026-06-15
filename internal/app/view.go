package app

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const reservedLines = 6 // margin, tabs, séparateur, blank, footer(2)

var stTabDanger = lipgloss.NewStyle().Bold(true).Foreground(cBlack).Background(cRed)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// renderTabs construit la barre d'onglets, avec fenêtre glissante si trop large.
func (m model) renderTabs(width int) string {
	labels := make([]string, len(m.pages))
	widths := make([]int, len(m.pages))
	for i, p := range m.pages {
		lab := strings.ToUpper(p.Title)
		if p.Icon != "" {
			lab = p.Icon + " " + lab
		}
		labels[i] = lab
		widths[i] = lipgloss.Width(lab)
	}

	const gap = 3
	target := width - 6
	if target < 12 {
		target = 12
	}

	start, end := m.pageIdx, m.pageIdx+1
	total := widths[m.pageIdx]
	for {
		grew := false
		if end < len(m.pages) && total+gap+widths[end] <= target {
			total += gap + widths[end]
			end++
			grew = true
		}
		if start > 0 && total+gap+widths[start-1] <= target {
			total += gap + widths[start-1]
			start--
			grew = true
		}
		if !grew {
			break
		}
	}

	activeStyle := stTabActive
	if m.focus == focusBar {
		activeStyle = stTabFocus
		if m.mode == modeDelConfirm && m.curPage().Kind == KindProjet {
			activeStyle = stTabDanger
		}
	}

	var parts []string
	for i := start; i < end; i++ {
		if i == m.pageIdx {
			parts = append(parts, activeStyle.Render(labels[i]))
		} else {
			parts = append(parts, stTabInactive.Render(labels[i]))
		}
	}
	bar := strings.Join(parts, strings.Repeat(" ", gap))

	res := "  " + bar
	if start > 0 {
		res = "  " + stDim.Render("‹") + " " + bar
	}
	if end < len(m.pages) {
		res += " " + stDim.Render("›")
	}
	return res
}

func emptyMessage(kind PageKind) string {
	switch kind {
	case KindFavoris:
		return "Aucun favori — a pour ajouter un chemin, f depuis un projet"
	case KindRecents:
		return "Aucun récent"
	default:
		return "Projet vide — a pour créer un dossier"
	}
}

func (m model) renderItem(i int, it Item, page Page) string {
	selected := i == m.selected && m.focus == focusList
	fav := ""
	if it.Fav && page.Kind != KindFavoris {
		fav = "  " + stFav.Render("★")
	}
	if selected && m.mode == modeDelConfirm {
		return "  " + stRed.Render("❯ "+it.Name) + fav
	}
	if selected {
		return "  " + stArrow.Render("❯") + " " + stSel.Render(it.Name) + fav
	}
	return "    " + stNormal.Render(it.Name) + fav
}

func (m model) View() string {
	width := m.width
	if width <= 0 {
		width = 80
	}
	page := m.curPage()

	var itemLines []string
	if len(page.Items) == 0 {
		itemLines = append(itemLines, "  "+stDim.Render(emptyMessage(page.Kind)))
	} else {
		for i, it := range page.Items {
			itemLines = append(itemLines, m.renderItem(i, it, page))
		}
	}

	avail := m.height - reservedLines
	if avail < 3 {
		avail = 3
	}
	n := len(itemLines)
	top := 0
	if n > avail {
		if m.focus == focusList && m.selected > avail-1 {
			top = m.selected - avail + 1
		}
		if top > n-avail {
			top = n - avail
		}
		if top < 0 {
			top = 0
		}
	}
	bot := top + avail
	if bot > n {
		bot = n
	}

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(m.renderTabs(width) + "\n")
	b.WriteString(stSep.Render("  "+strings.Repeat("─", min(width-4, 60))) + "\n")
	b.WriteString("\n")
	for i := top; i < bot; i++ {
		b.WriteString(itemLines[i] + "\n")
	}
	for pad := bot - top; pad < avail; pad++ {
		b.WriteString("\n")
	}
	b.WriteString(m.footer())
	return b.String()
}

func inputLabel(action string) string {
	switch action {
	case "favPath":
		return "Chemin du dossier"
	case "newProjet":
		return "Chemin du projet"
	case "newDir":
		return "Nom du dossier"
	case "rename":
		return "Renommer en"
	}
	return ""
}

func (m model) delPrompt() string {
	page := m.curPage()
	if m.focus == focusBar && page.Kind == KindProjet {
		return "Retirer le projet " + stSel.Render(page.Title) + " ?"
	}
	it, _ := m.current()
	switch page.Kind {
	case KindFavoris:
		return "Retirer le favori " + stSel.Render(it.Name) + " ?"
	case KindProjet:
		return "Mettre " + stSel.Render(it.Name) + " à la corbeille ?"
	}
	return "Supprimer ?"
}

func (m model) footer() string {
	switch m.mode {
	case modeInput:
		return "\n  " + stArrow.Render("›") + " " + inputLabel(m.inputAction) + " : " + m.input.View()
	case modeDelConfirm:
		return "\n  " + stArrow.Render("›") + " " + m.delPrompt() + " " + stKey.Render("(o/n)")
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
	var parts []string
	page := m.curPage()

	if m.focus == focusBar {
		parts = append(parts, key("←→", "onglet"), key("↓", "entrer"), key("a", "projet"))
		if page.Kind == KindProjet {
			parts = append(parts, key("s", "retirer"), key("r", "renommer"), key("⏎", "finder"))
		}
		parts = append(parts, key("q", "quit"))
		return strings.Join(parts, "  ")
	}

	parts = append(parts, key("↑↓", ""), key("←→", "onglet"), key("⏎", "vscode"), key("c", "claude"), key("x", "codex"))
	switch page.Kind {
	case KindFavoris:
		parts = append(parts, key("a", "path"), key("s", "retirer"))
	case KindProjet:
		parts = append(parts, key("a", "dossier"), key("s", "suppr"), key("r", "renommer"), key("f", "favori"))
	case KindRecents:
		parts = append(parts, key("f", "favori"))
	}
	parts = append(parts, key("q", "quit"))
	return strings.Join(parts, "  ")
}
