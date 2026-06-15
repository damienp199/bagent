package main

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// --- Styles ---
var (
	cOrange = lipgloss.Color("208")
	cWhite  = lipgloss.Color("231")
	cGray   = lipgloss.Color("243")
	cGreen  = lipgloss.Color("107")
	cRed    = lipgloss.Color("167")

	stTitle    = lipgloss.NewStyle().Bold(true).Foreground(cWhite)
	stPuce     = lipgloss.NewStyle().Foreground(cOrange)
	stSection  = lipgloss.NewStyle().Bold(true).Foreground(cGray)
	stArrow    = lipgloss.NewStyle().Foreground(cOrange)
	stSel      = lipgloss.NewStyle().Bold(true).Foreground(cWhite)
	stNormal   = lipgloss.NewStyle().Foreground(cGray)
	stDim      = lipgloss.NewStyle().Faint(true)
	stDefault  = lipgloss.NewStyle().Faint(true).Foreground(cOrange)
	stGreen    = lipgloss.NewStyle().Foreground(cGreen)
	stRed      = lipgloss.NewStyle().Foreground(cRed)
	stFooter   = lipgloss.NewStyle().Foreground(cGray)
	stKey      = lipgloss.NewStyle().Faint(true)
)

type uiMode int

const (
	modeList uiMode = iota
	modeAddPath
	modeAddKind
	modeDelConfirm
)

type clearStatusMsg struct{}

type model struct {
	items       []Item
	recentStart int
	selected    int
	scrollTop   int
	width       int
	height      int

	mode        uiMode
	input       textinput.Model
	pendingPath string
	status      string

	// résultat à exécuter après la sortie
	action string // "claude" | "codex" | "vscode"
	target string
}

func newModel() model {
	ti := textinput.New()
	ti.Prompt = ""
	ti.CharLimit = 0
	ti.Width = 50

	m := model{mode: modeList, input: ti, height: 24, width: 80}
	m.reload()
	return m
}

func (m *model) reload() {
	m.items, m.recentStart = buildItems()
	m.fixSelected()
	m.recomputeScroll()
}

func (m *model) fixSelected() {
	if m.selected < 0 {
		m.selected = 0
	}
	if m.selected > len(m.items)-1 {
		m.selected = len(m.items) - 1
	}
	// Sauter les headers de groupe (non sélectionnables) vers le bas.
	for m.selected >= 0 && m.selected < len(m.items) && m.items[m.selected].Type == TypeGroupHeader {
		m.selected++
	}
	if m.selected > len(m.items)-1 {
		m.selected = len(m.items) - 1
	}
	if m.selected < 0 {
		m.selected = 0
	}
}

func (m model) Init() tea.Cmd { return nil }

func clearStatusCmd() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg { return clearStatusMsg{} })
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.recomputeScroll()
		return m, nil
	case clearStatusMsg:
		m.status = ""
		return m, nil
	case tea.KeyMsg:
		switch m.mode {
		case modeAddPath:
			return m.updateAddPath(msg)
		case modeAddKind:
			return m.updateAddKind(msg)
		case modeDelConfirm:
			return m.updateDelConfirm(msg)
		default:
			return m.updateList(msg)
		}
	}
	return m, nil
}

func (m *model) current() (Item, bool) {
	if len(m.items) == 0 || m.selected < 0 || m.selected >= len(m.items) {
		return Item{}, false
	}
	return m.items[m.selected], true
}

func (m *model) moveUp() {
	prev := m.selected - 1
	for prev >= 0 && m.items[prev].Type == TypeGroupHeader {
		prev--
	}
	if prev >= 0 {
		m.selected = prev
	}
	m.recomputeScroll()
}

func (m *model) moveDown() {
	next := m.selected + 1
	for next < len(m.items) && m.items[next].Type == TypeGroupHeader {
		next++
	}
	if next < len(m.items) {
		m.selected = next
	}
	m.recomputeScroll()
}

func (m model) launch(action string, it Item) (tea.Model, tea.Cmd) {
	if !isDir(it.FullPath) {
		m.status = stRed.Render("✗") + " Dossier introuvable"
		return m, clearStatusCmd()
	}
	switch action {
	case "claude", "codex":
		if !toolAvailable(action) {
			m.status = stRed.Render("✗") + " " + action + " non installé"
			return m, clearStatusCmd()
		}
	}
	m.action = action
	m.target = it.FullPath
	return m, tea.Quit
}

func (m model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "up", "k":
		m.moveUp()
	case "down", "j":
		m.moveDown()
	case "enter":
		if it, ok := m.current(); ok {
			return m.launch("vscode", it)
		}
	case "c":
		if it, ok := m.current(); ok {
			return m.launch("claude", it)
		}
	case "x":
		if it, ok := m.current(); ok {
			return m.launch("codex", it)
		}
	case "a":
		if it, ok := m.current(); ok && it.Type == TypeRecent {
			// Épingler directement un récent.
			if added, err := addEntry(it.FullPath); err == nil && added {
				m.status = stGreen.Render("✓") + " Épinglé " + stDim.Render(filepath.Base(it.FullPath))
				m.selected = 0
				m.reload()
				return m, clearStatusCmd()
			}
		}
		m.input.SetValue("")
		m.input.Focus()
		m.mode = modeAddPath
		return m, textinput.Blink
	case "s":
		if it, ok := m.current(); ok && (it.Type == TypeWorkspace || it.Type == TypeGroupItem) {
			m.mode = modeDelConfirm
		}
	case "d":
		if it, ok := m.current(); ok && it.Type != TypeGroupHeader {
			entry := it.FullPath
			if it.Type == TypeGroupItem {
				entry = ">" + it.Group
			}
			if err := setDefault(entry); err == nil {
				m.status = stGreen.Render("✓") + " Défaut → " + it.Name
				m.selected = 0
				m.reload()
				return m, clearStatusCmd()
			}
		}
	default:
		// Raccourcis numériques 1-9 → ouverture VSCode (action par défaut).
		s := msg.String()
		if len(s) == 1 && s[0] >= '1' && s[0] <= '9' {
			n := int(s[0] - '0')
			for _, it := range m.items {
				if it.Num == n {
					return m.launch("vscode", it)
				}
			}
		}
	}
	return m, nil
}

func (m model) updateAddPath(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeList
		m.input.Blur()
		return m, nil
	case "enter":
		p := cleanPath(m.input.Value())
		m.input.Blur()
		if p == "" {
			m.mode = modeList
			return m, nil
		}
		if !isDir(p) {
			m.mode = modeList
			m.status = stRed.Render("✗") + " Dossier introuvable"
			return m, clearStatusCmd()
		}
		m.pendingPath = p
		m.mode = modeAddKind
		return m, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m model) updateAddKind(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeList
		m.pendingPath = ""
		return m, nil
	case "p", "P":
		entry := ">" + m.pendingPath
		added, _ := addEntry(entry)
		if added {
			m.status = stGreen.Render("✓") + " Groupe " + filepath.Base(m.pendingPath)
		} else {
			m.status = stRed.Render("✗") + " Déjà dans la liste"
		}
		m.pendingPath = ""
		m.mode = modeList
		m.selected = 0
		m.reload()
		return m, clearStatusCmd()
	case "d", "D", "enter":
		added, _ := addEntry(m.pendingPath)
		if added {
			m.status = stGreen.Render("✓") + " " + filepath.Base(m.pendingPath)
		} else {
			m.status = stRed.Render("✗") + " Déjà dans la liste"
		}
		m.pendingPath = ""
		m.mode = modeList
		m.selected = 0
		m.reload()
		return m, clearStatusCmd()
	}
	return m, nil
}

func (m model) updateDelConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "o", "O":
		it, ok := m.current()
		if !ok {
			m.mode = modeList
			return m, nil
		}
		if it.Type == TypeWorkspace {
			_ = removeEntry(it.FullPath)
			m.status = stGreen.Render("✓") + " Supprimé " + stDim.Render(it.Name)
		} else if it.Type == TypeGroupItem {
			_ = removeEntry(">" + it.Group)
			m.status = stGreen.Render("✓") + " Supprimé " + stDim.Render(filepath.Base(it.Group))
		}
		m.mode = modeList
		m.reload()
		return m, clearStatusCmd()
	default:
		m.mode = modeList
		return m, nil
	}
}

// cleanPath nettoie un chemin saisi (espaces, quotes, slash final).
func cleanPath(p string) string {
	p = strings.TrimSpace(p)
	p = strings.Trim(p, "'\"")
	p = strings.TrimSpace(p)
	p = strings.TrimSuffix(p, "/")
	return p
}
