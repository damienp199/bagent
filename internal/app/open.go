package app

import (
	"fmt"
	"os"
	"strings"
)

// parseOpenArgs extrait l'action execAction (vscode/claude/codex) et le chemin
// cible des arguments de `bagent open`. Défaut : vscode. `--tool=code` est un
// alias de vscode.
func parseOpenArgs(args []string) (action, target string) {
	action = "vscode"
	for _, a := range args {
		switch {
		case strings.HasPrefix(a, "--tool="):
			tool := strings.TrimPrefix(a, "--tool=")
			if tool == "code" {
				tool = "vscode"
			}
			action = tool
		case !strings.HasPrefix(a, "-"):
			target = a
		}
	}
	return action, target
}

// runOpen ouvre un workspace dans l'outil demandé. Colle appelée par le
// workflow Alfred (actions Run Script). Pour claude/codex, on ouvre une
// nouvelle fenêtre de terminal (openTerminalWith) : depuis Alfred il n'y a pas
// de terminal courant, donc l'exec in-place de execAction ne s'applique pas.
func runOpen(args []string) {
	action, target := parseOpenArgs(args)
	if target == "" {
		fmt.Fprintln(os.Stderr, "  ✗ chemin manquant")
		os.Exit(1)
	}
	switch action {
	case "claude", "codex":
		if err := openTerminalWith(action, target); err != nil {
			fmt.Fprintln(os.Stderr, "  ✗", err)
			os.Exit(1)
		}
	default:
		execAction(action, target)
	}
}
