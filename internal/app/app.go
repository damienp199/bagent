package app

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func Run() {
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

// runDefault ouvre le 1er favori (sinon le 1er workspace disponible)
// dans VSCode, l'action par défaut.
func runDefault() {
	pages := buildPages()
	target := ""
	// Priorité aux favoris.
	if fi := favorisIndex(pages); len(pages[fi].Items) > 0 {
		target = pages[fi].Items[0].FullPath
	}
	if target == "" {
		for _, p := range pages {
			if len(p.Items) > 0 {
				target = p.Items[0].FullPath
				break
			}
		}
	}
	if target == "" {
		fmt.Fprintln(os.Stderr, "  ✗ Aucun workspace configuré")
		os.Exit(1)
	}
	if !isDir(target) {
		fmt.Fprintln(os.Stderr, "  ✗ Dossier introuvable :", target)
		os.Exit(1)
	}
	if err := openVSCode(target); err != nil {
		fmt.Fprintln(os.Stderr, "  ✗", err)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println(`  bagent — lanceur de workspaces

  Usage:
    bagent          Ouvrir le menu
    bagent -d       Ouvrir le premier workspace (VSCode)
    bagent --help   Afficher cette aide

  Onglets : Favoris · Projets · Récents

  Navigation:
    ←→ / h l   changer d'onglet
    ↑↓ / k j   naviguer ; ↑ depuis le 1er item remonte sur la barre
    ⏎ c x      ouvrir (VSCode / claude / codex)

  Sur la barre d'onglets:
    a          nouveau projet (chemin du dossier)
    s          retirer le projet      o  réordonner les onglets
    ⏎          ouvrir le dossier dans le Finder

  Dans un projet:
    a          créer un dossier       f  ajouter aux favoris

  Dans Favoris:
    a          ajouter un chemin      f  retirer le favori

  q            quitter`)
}
