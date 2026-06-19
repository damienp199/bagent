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

// refreshInterval : période de relecture du disque pour refléter les
// changements externes (dossiers créés/supprimés depuis le Finder ou un autre
// terminal) sans relancer l'app.
const refreshInterval = 1500 * time.Millisecond

// --- Styles ---
var (
	cOrange = lipgloss.Color("208")
	cGray   = lipgloss.Color("243")
	cGreen  = lipgloss.Color("107")
	cRed    = lipgloss.Color("167")
	cBlack  = lipgloss.Color("16")

	// Texte de l'item sélectionné : noir sur terminal clair, blanc sur terminal
	// sombre. Évite le texte blanc invisible sur fond clair.
	cSel = lipgloss.AdaptiveColor{Light: "16", Dark: "231"}

	stTabActive   = lipgloss.NewStyle().Bold(true).Foreground(cOrange).Padding(0, 1)
	stTabFocus    = lipgloss.NewStyle().Bold(true).Foreground(cBlack).Background(cOrange).Padding(0, 1)
	stTabInactive = lipgloss.NewStyle().Foreground(cGray).Padding(0, 1)
	stArrow       = lipgloss.NewStyle().Foreground(cOrange)
	stSel         = lipgloss.NewStyle().Bold(true).Foreground(cSel)
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
	modeReorder
)

type focusZone int

const (
	focusList focusZone = iota
	focusBar
)

type clearStatusMsg struct{}

type refreshMsg struct{}

type model struct {
	pages    []Page
	pageIdx  int
	selected int
	focus    focusZone
	width    int
	height   int

	mode        uiMode
	inputAction string // "favPath" | "newProjet" | "newDir"
	input       textinput.Model
	status      string

	action string // résultat à exécuter après la sortie
	target string

	updateTag string // tag d'une mise à jour disponible ("" sinon)
}

func newModel() model {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Width = 50
	m := model{mode: modeList, focus: focusList, input: ti, height: 24, width: 80}
	pruneDeadEntries() // purge les projets pointant vers un dossier disparu
	m.pages = buildPages()
	m.pageIdx = favorisIndex(m.pages) // page d'accueil = Favoris
	m.clamp()
	return m
}

func (m *model) reload() {
	m.pages = buildPages()
	m.clamp()
}

// locatePage renvoie l'index de la page identifiée par (kind, parent), ou -1.
func locatePage(pages []Page, kind PageKind, parent string) int {
	for i, p := range pages {
		if p.Kind == kind && p.Parent == parent {
			return i
		}
	}
	return -1
}

// reapplyPages remplace les pages en préservant autant que possible la position
// de l'utilisateur : l'onglet courant est suivi par identité (nature + parent)
// et l'item sélectionné par chemin. Repli sur Favoris si l'onglet a disparu ;
// sélection clampée si l'item a disparu.
func (m *model) reapplyPages(pages []Page) {
	prevKind := m.curPage().Kind
	prevParent := m.curPage().Parent
	prevPath := ""
	if it, ok := m.current(); ok {
		prevPath = it.FullPath
	}

	m.pages = pages
	m.pageIdx = locatePage(pages, prevKind, prevParent)
	if m.pageIdx < 0 {
		m.pageIdx = favorisIndex(pages)
	}
	m.selected = 0
	if prevPath != "" {
		m.selectByPath(prevPath)
	}
	m.clamp()
}

// reloadPreserving relit le disque et réapplique les pages sans perdre la
// position courante.
func (m *model) reloadPreserving() { m.reapplyPages(buildPages()) }

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

func (m model) Init() tea.Cmd { return tea.Batch(refreshCmd(), updateCheckCmd()) }

func clearStatusCmd() tea.Cmd {
	return tea.Tick(statusDelay, func(t time.Time) tea.Msg { return clearStatusMsg{} })
}

func refreshCmd() tea.Cmd {
	return tea.Tick(refreshInterval, func(t time.Time) tea.Msg { return refreshMsg{} })
}

type updateAvailableMsg struct{ tag string }

func updateCheckCmd() tea.Cmd {
	return func() tea.Msg {
		return updateAvailableMsg{tag: checkForUpdate(time.Now().Unix(), httpFetch)}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case clearStatusMsg:
		m.status = ""
		return m, nil
	case refreshMsg:
		// Reflète les changements externes, mais jamais pendant une saisie,
		// un réordonnancement ou une confirmation (positions volatiles).
		if m.mode == modeList {
			m.reloadPreserving()
		}
		return m, refreshCmd()
	case updateAvailableMsg:
		m.updateTag = msg.tag
		return m, nil
	case tea.KeyMsg:
		switch m.mode {
		case modeInput:
			return m.updateInput(msg)
		case modeDelConfirm:
			return m.updateDelConfirm(msg)
		case modeReorder:
			return m.updateReorder(msg)
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
	if action == "vscode" {
		// VSCode s'ouvre en détaché : on reste dans le menu (pas de tea.Quit).
		if err := openVSCode(it.FullPath); err != nil {
			m.status = stRed.Render("✗") + " " + err.Error()
		} else {
			m.status = stGreen.Render("✓") + " VSCode " + stDim.Render(it.Name)
		}
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
		// Toujours descendre dans la liste, même vide : c'est là qu'on ajoute
		// un item (a) au groupe courant. Bloquer ici piégeait l'utilisateur sur
		// la barre, où a crée un projet au lieu d'un item.
		m.focus = focusList
		m.selected = 0
		return m, nil
	case "a":
		return m, m.startInput("newProjet", "")
	}
	if page.Kind == KindProjet {
		switch s {
		case "o":
			m.mode = modeReorder
			return m, nil
		case "s":
			m.mode = modeDelConfirm
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

// updateReorder : mode « ordre ». Sur la liste des favoris, ↑/↓ déplacent le
// favori sélectionné ; sur la barre, ←/→ déplacent l'onglet projet courant.
func (m model) updateReorder(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	s := msg.String()
	switch s {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "esc", "enter", "o":
		m.mode = modeList
		return m, nil
	}
	if m.focus == focusList && m.curPage().Kind == KindFavoris {
		switch s {
		case "up":
			return m.moveCurrentFavorite(-1)
		case "down":
			return m.moveCurrentFavorite(+1)
		}
		return m, nil
	}
	switch s {
	case "left", "h":
		return m.moveCurrentProject(-1)
	case "right", "l":
		return m.moveCurrentProject(+1)
	}
	return m, nil
}

// moveCurrentFavorite déplace le favori sélectionné d'une position et fait
// suivre la sélection.
func (m model) moveCurrentFavorite(dir int) (tea.Model, tea.Cmd) {
	items := m.curPage().Items
	j := m.selected + dir
	if j < 0 || j >= len(items) {
		return m, nil // bord (haut/bas de liste)
	}
	cur := items[m.selected].FullPath
	if swapFavorites(cur, items[j].FullPath) {
		m.reload()
		m.selectByPath(cur)
	}
	return m, nil
}

// moveCurrentProject déplace l'onglet projet courant d'une position et fait
// suivre le focus au projet déplacé.
func (m model) moveCurrentProject(dir int) (tea.Model, tea.Cmd) {
	j := m.pageIdx + dir
	if j < 0 || j >= len(m.pages) || m.pages[j].Kind != KindProjet {
		return m, nil // bord visuel (Favoris à gauche, dernier onglet à droite)
	}
	parent := m.curPage().Parent
	if swapProjects(parent, m.pages[j].Parent) {
		m.reload()
		m.gotoProjet(parent)
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
		if it, ok := m.current(); ok {
			if page.Kind == KindFavoris {
				_ = removeFavorite(it.FullPath)
				m.status = stGreen.Render("✓") + " Favori retiré " + stDim.Render(it.Name)
			} else {
				_ = toggleFavorite(it.FullPath)
				m.status = stFav.Render("★") + " Favori + " + stDim.Render(it.Name)
			}
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
		case "o":
			if len(page.Items) > 0 {
				m.mode = modeReorder
			}
			return m, nil
		}
	case KindProjet:
		if s == "a" {
			return m, m.startInput("newDir", "")
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
	}
	return m, nil
}

func (m model) updateDelConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if s := msg.String(); s == "o" || s == "O" {
		page := m.curPage()
		if m.focus == focusBar && page.Kind == KindProjet {
			_ = removeEntry(">" + page.Parent)
			m.status = stGreen.Render("✓") + " Projet retiré " + stDim.Render(page.Title)
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
