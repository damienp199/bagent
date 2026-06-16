package app

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func stripANSI(s string) string {
	var b strings.Builder
	esc := false
	for _, r := range s {
		if r == '\x1b' {
			esc = true
			continue
		}
		if esc {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				esc = false
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func press(m model, s string) model {
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	switch s {
	case "right":
		msg = tea.KeyMsg{Type: tea.KeyRight}
	case "left":
		msg = tea.KeyMsg{Type: tea.KeyLeft}
	case "up":
		msg = tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		msg = tea.KeyMsg{Type: tea.KeyDown}
	case "enter":
		msg = tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		msg = tea.KeyMsg{Type: tea.KeyEsc}
	}
	nm, _ := m.Update(msg)
	return nm.(model)
}

func TestNavigateToPopulatedPage(t *testing.T) {
	m := newModel()
	m.width, m.height = 100, 30
	for i := 0; i < len(m.pages); i++ {
		if len(m.curPage().Items) > 0 && m.curPage().Kind == KindProjet {
			out := stripANSI(m.View())
			first := m.curPage().Items[0].Name
			if !strings.Contains(out, first) {
				t.Errorf("page %q ne montre pas l'item %q :\n%s", m.curPage().Title, first, out)
			}
			t.Logf("page peuplée: %s (%d items)\n%s", m.curPage().Title, len(m.curPage().Items), out)
			return
		}
		m = press(m, "right")
	}
	t.Skip("aucune page de groupe peuplée")
}
