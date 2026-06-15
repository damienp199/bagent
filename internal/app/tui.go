package app

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const statusDelay = 1500 * time.Millisecond

// --- Styles ---
var (
	cOrange = lipgloss.Color("208")
	cWhite  = lipgloss.Color("231")
	cGray   = lipgloss.Color("243")
	cGreen  = lipgloss.Color("107")
	cRed    = lipgloss.Color("167")
	cBlack  = lipgloss.Color("16")

	stTabActive   = lipgloss.NewStyle().Bold(true).Foreground(cOrange)
	stTabFocus    = lipgloss.NewStyle().Bold(true).Foreground(cBlack).Background(cOrange)
	stTabInactive = lipgloss.NewStyle().Foreground(cGray)
	stArrow       = lipgloss.NewStyle().Foreground(cOrange)
	stSel         = lipgloss.NewStyle().Bold(true).Foreground(cWhite)
	stNormal      = lipgloss.NewStyle().Foreground(cGray)
	stFav         = lipgloss.NewStyle().Foreground(cOrange)
	stDim         = lipgloss.NewStyle().Faint(true)
	stGreen       = lipgloss.NewStyle().Foreground(cGreen)
	stRed         = lipgloss.NewStyle().Foreground(cRed)
	stFooter      = lipgloss.NewStyle().Foreground(cGray)
	stKey         = lipgloss.NewStyle().Faint(true)
	stSep         = lipgloss.NewStyle().Foreground(cGray).Faint(true)
)

type uiMode int

const (
	modeList uiMode = iota
	modeInput
	modeDelConfirm
)

type focusZone int

const (
	focusList focusZone = iota
	focusBar
)

type clearStatusMsg struct{}

type model struct {
	pages    []Page
	pageIdx  int
	selected int
	focus    focusZone
	width    int
	height   int

	mode        uiMode
	inputAction string // "favPath" | "newProjet" | "newDir" | "rename"
	input       textinput.Model
	status      string

	action string // résultat à exécuter après la sortie
	target string
}

func newModel() model {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Width = 50
	m := model{mode: modeList, focus: focusList, input: ti, height: 24, width: 80}
	m.pages = buildPages()
	m.clamp()
	return m
}

func (m *model) reload() {
	m.pages = buildPages()
	m.clamp()
}

func (m *model) clamp() {
	if len(m.pages) == 0 {
		return
	}
	if m.pageIdx < 0 {
		m.pageIdx = 0
	}
	if m.pageIdx > len(m.pages)-1 {
		m.pageIdx = len(m.pages) - 1
	}
	items := m.curPage().Items
	if m.selected > len(items)-1 {
		m.selected = len(items) - 1
	}
	if m.selected < 0 {
		m.selected = 0
	}
}

func (m model) curPage() Page {
	if m.pageIdx < 0 || m.pageIdx >= len(m.pages) {
		return Page{}
	}
	return m.pages[m.pageIdx]
}

func (m *model) current() (Item, bool) {
	items := m.curPage().Items
	if m.selected < 0 || m.selected >= len(items) {
		return Item{}, false
	}
	return items[m.selected], true
}

func (m *model) nextPage() {
	if len(m.pages) == 0 {
		return
	}
	m.pageIdx = (m.pageIdx + 1) % len(m.pages)
	m.selected = 0
}

func (m *model) prevPage() {
	if len(m.pages) == 0 {
		return
	}
	m.pageIdx = (m.pageIdx - 1 + len(m.pages)) % len(m.pages)
	m.selected = 0
}

func (m *model) gotoProjet(parent string) {
	for i, p := range m.pages {
		if p.Kind == KindProjet && p.Parent == parent {
			m.pageIdx = i
			return
		}
	}
}

func (m *model) selectByPath(path string) {
	for i, it := range m.curPage().Items {
		if it.FullPath == path {
			m.selected = i
			return
		}
	}
}

func (m *model) startInput(action, val string) tea.Cmd {
	m.inputAction = action
	m.input.SetValue(val)
	m.input.CursorEnd()
	m.input.Focus()
	m.mode = modeInput
	return textinput.Blink
}

func (m model) Init() tea.Cmd { return nil }

func clearStatusCmd() tea.Cmd {
	return tea.Tick(statusDelay, func(t time.Time) tea.Msg { return clearStatusMsg{} })
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case clearStatusMsg:
		m.status = ""
		return m, nil
	case tea.KeyMsg:
		switch m.mode {
		case modeInput:
			return m.updateInput(msg)
		case modeDelConfirm:
			return m.updateDelConfirm(msg)
		default:
			return m.updateList(msg)
		}
	}
	return m, nil
}

func (m model) launch(action string, it Item) (tea.Model, tea.Cmd) {
	if !isDir(it.FullPath) {
		m.status = stRed.Render("✗") + " Dossier introuvable"
		return m, clearStatusCmd()
	}
	if action == "claude" || action == "codex" {
		if !toolAvailable(action) {
			m.status = stRed.Render("✗") + " " + action + " non installé"
			return m, clearStatusCmd()
		}
	}
	m.action, m.target = action, it.FullPath
	return m, tea.Quit
}

func (m model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	s := msg.String()
	switch s {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "left", "h":
		m.prevPage()
		return m, nil
	case "right", "l":
		m.nextPage()
		return m, nil
	}
	if m.focus == focusBar {
		return m.updateBar(s)
	}
	return m.updateItems(s)
}

// updateBar : focus sur la barre d'onglets (gestion des projets).
func (m model) updateBar(s string) (tea.Model, tea.Cmd) {
	page := m.curPage()
	switch s {
	case "down", "j":
		if len(page.Items) > 0 {
			m.focus = focusList
			m.selected = 0
		}
		return m, nil
	case "a":
		return m, m.startInput("newProjet", "")
	}
	if page.Kind == KindProjet {
		switch s {
		case "s":
			m.mode = modeDelConfirm
		case "r":
			return m, m.startInput("rename", page.Title)
		case "enter":
			if err := openFinder(page.Parent); err != nil {
				m.status = stRed.Render("✗") + " " + err.Error()
			} else {
				m.status = stGreen.Render("✓") + " Finder " + stDim.Render(page.Title)
			}
			return m, clearStatusCmd()
		}
	}
	return m, nil
}

// updateItems : focus sur la liste (gestion des items).
func (m model) updateItems(s string) (tea.Model, tea.Cmd) {
	page := m.curPage()
	switch s {
	case "up", "k":
		if m.selected > 0 {
			m.selected--
		} else {
			m.focus = focusBar
		}
		return m, nil
	case "down", "j":
		if m.selected < len(page.Items)-1 {
			m.selected++
		}
		return m, nil
	case "enter":
		if it, ok := m.current(); ok {
			return m.launch("vscode", it)
		}
		return m, nil
	case "c":
		if it, ok := m.current(); ok {
			return m.launch("claude", it)
		}
		return m, nil
	case "x":
		if it, ok := m.current(); ok {
			return m.launch("codex", it)
		}
		return m, nil
	case "f":
		if it, ok := m.current(); ok && page.Kind != KindFavoris {
			_ = toggleFavorite(it.FullPath)
			m.status = stFav.Render("★") + " Favori + " + stDim.Render(it.Name)
			m.reload()
			return m, clearStatusCmd()
		}
		return m, nil
	}
	switch page.Kind {
	case KindFavoris:
		switch s {
		case "a":
			return m, m.startInput("favPath", "")
		case "s":
			if _, ok := m.current(); ok {
				m.mode = modeDelConfirm
			}
		}
	case KindProjet:
		switch s {
		case "a":
			return m, m.startInput("newDir", "")
		case "s":
			if _, ok := m.current(); ok {
				m.mode = modeDelConfirm
			}
		case "r":
			if it, ok := m.current(); ok {
				return m, m.startInput("rename", it.Name)
			}
		}
	}
	return m, nil
}

func (m model) updateInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeList
		m.input.Blur()
		return m, nil
	case "enter":
		val := cleanPath(m.input.Value())
		m.input.Blur()
		m.mode = modeList
		return m.applyInput(val)
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m model) applyInput(val string) (tea.Model, tea.Cmd) {
	if val == "" {
		return m, nil
	}
	switch m.inputAction {
	case "favPath":
		if !isDir(val) {
			m.status = stRed.Render("✗") + " Dossier introuvable"
			return m, clearStatusCmd()
		}
		added, _ := addFavorite(val)
		if added {
			m.status = stFav.Render("★") + " Favori + " + stDim.Render(filepath.Base(val))
		} else {
			m.status = stRed.Render("✗") + " Déjà en favori"
		}
		m.reload()
		m.selectByPath(val)
		return m, clearStatusCmd()

	case "newProjet":
		if !isDir(val) {
			m.status = stRed.Render("✗") + " Dossier introuvable"
			return m, clearStatusCmd()
		}
		added, _ := addEntry(">" + val)
		if added {
			m.status = stGreen.Render("✓") + " Projet " + stDim.Render(filepath.Base(val))
		} else {
			m.status = stRed.Render("✗") + " Déjà un projet"
		}
		m.reload()
		m.gotoProjet(val)
		m.focus = focusBar
		return m, clearStatusCmd()

	case "newDir":
		parent := m.curPage().Parent
		full, err := createDir(parent, val)
		switch {
		case err == os.ErrExist:
			m.status = stRed.Render("✗") + " Existe déjà " + stDim.Render(val)
		case err != nil:
			m.status = stRed.Render("✗") + " " + err.Error()
		default:
			m.status = stGreen.Render("✓") + " Créé " + stDim.Render(val)
		}
		m.reload()
		m.focus = focusList
		m.selectByPath(full)
		return m, clearStatusCmd()

	case "rename":
		return m.applyRename(val)
	}
	return m, nil
}

func (m model) applyRename(val string) (tea.Model, tea.Cmd) {
	// Renommage d'un projet (focus barre) ou d'un dossier (focus liste).
	old := ""
	isProjet := m.focus == focusBar && m.curPage().Kind == KindProjet
	if isProjet {
		old = m.curPage().Parent
	} else if it, ok := m.current(); ok {
		old = it.FullPath
	} else {
		return m, nil
	}
	newPath, err := renameDir(old, val)
	switch {
	case err == os.ErrExist:
		m.status = stRed.Render("✗") + " Existe déjà " + stDim.Render(val)
	case err != nil:
		m.status = stRed.Render("✗") + " " + err.Error()
	default:
		m.status = stGreen.Render("✓") + " Renommé " + stDim.Render(val)
	}
	m.reload()
	if isProjet {
		m.gotoProjet(newPath)
		m.focus = focusBar
	} else {
		m.selectByPath(newPath)
	}
	return m, clearStatusCmd()
}

func (m model) updateDelConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if s := msg.String(); s == "o" || s == "O" {
		page := m.curPage()
		if m.focus == focusBar && page.Kind == KindProjet {
			_ = removeEntry(">" + page.Parent)
			m.status = stGreen.Render("✓") + " Projet retiré " + stDim.Render(page.Title)
		} else if it, ok := m.current(); ok {
			switch page.Kind {
			case KindFavoris:
				_ = removeFavorite(it.FullPath)
				m.status = stGreen.Render("✓") + " Retiré " + stDim.Render(it.Name)
			case KindProjet:
				if err := trashDir(it.FullPath); err != nil {
					m.status = stRed.Render("✗") + " " + err.Error()
				} else {
					m.status = stGreen.Render("✓") + " Corbeille " + stDim.Render(it.Name)
				}
			}
		}
		m.mode = modeList
		m.reload()
		return m, clearStatusCmd()
	}
	m.mode = modeList
	return m, nil
}

// cleanPath nettoie une saisie (espaces, quotes, slash final).
func cleanPath(p string) string {
	p = strings.TrimSpace(p)
	p = strings.Trim(p, "'\"")
	p = strings.TrimSpace(p)
	p = strings.TrimSuffix(p, "/")
	return p
}
