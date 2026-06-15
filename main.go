package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--help", "-h":
			printHelp()
			return
		case "-d":
			runDefault()
			return
		}
	}

	p := tea.NewProgram(newModel(), tea.WithAltScreen())
	res, err := p.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	m, ok := res.(model)
	if !ok {
		return
	}
	execAction(m.action, m.target)
}

func execAction(action, target string) {
	switch action {
	case "vscode":
		if err := openVSCode(target); err != nil {
			fmt.Fprintln(os.Stderr, "  ✗", err)
			os.Exit(1)
		}
	case "claude", "codex":
		// Remplace le process courant ; ne revient qu'en cas d'erreur.
		if err := runInTerminal(action, target); err != nil {
			fmt.Fprintln(os.Stderr, "  ✗", err)
			os.Exit(1)
		}
	}
}

// runDefault ouvre le workspace par défaut (1re ligne) dans le terminal avec claude.
func runDefault() {
	items, _ := buildItems()
	if len(items) == 0 {
		fmt.Fprintln(os.Stderr, "  ✗ Aucun workspace configuré")
		os.Exit(1)
	}
	idx := 0
	for idx < len(items) && items[idx].Type == TypeGroupHeader {
		idx++
	}
	if idx >= len(items) {
		fmt.Fprintln(os.Stderr, "  ✗ Aucun workspace configuré")
		os.Exit(1)
	}
	target := items[idx].FullPath
	if !isDir(target) {
		fmt.Fprintln(os.Stderr, "  ✗ Dossier introuvable :", target)
		os.Exit(1)
	}
	if err := runInTerminal("claude", target); err != nil {
		fmt.Fprintln(os.Stderr, "  ✗", err)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println(`  bagent — lanceur de workspaces

  Usage:
    bagent          Ouvrir le menu
    bagent -d       Ouvrir le workspace par défaut (claude)
    bagent --help   Afficher cette aide

  Dans le menu:
    ↑↓ / j k   naviguer
    ⏎          ouvrir dans VSCode
    c          ouvrir avec claude
    x          ouvrir avec codex
    1-9        ouvrir par numéro (VSCode)
    a          ajouter un workspace
    s          supprimer
    d          définir par défaut
    q          quitter`)
}
