package app

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

var pathEnriched bool

// ensurePATH complète le PATH du process avec celui du shell de login interactif.
// Nécessaire car ~/.local/bin (claude, codex…) n'est ajouté que dans .zshrc :
// si bagent est lancé hors d'un shell interactif, ces binaires sont invisibles.
// Appelé paresseusement (seulement si un binaire est introuvable).
func ensurePATH() {
	if pathEnriched {
		return
	}
	pathEnriched = true
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/zsh"
	}
	out, err := exec.Command(shell, "-lic", `printf "BAGENTPATH:%s:END" "$PATH"`).Output()
	if err != nil {
		return
	}
	s := string(out)
	i := strings.Index(s, "BAGENTPATH:")
	j := strings.Index(s, ":END")
	if i >= 0 && j > i {
		if p := s[i+len("BAGENTPATH:") : j]; p != "" {
			os.Setenv("PATH", p)
		}
	}
}

// toolAvailable indique si un exécutable est présent dans le PATH (en
// complétant le PATH via le shell de login si la première recherche échoue).
func toolAvailable(bin string) bool {
	if _, err := exec.LookPath(bin); err == nil {
		return true
	}
	ensurePATH()
	_, err := exec.LookPath(bin)
	return err == nil
}

// runInTerminal remplace le process courant par la commande, dans le workspace.
// Réutilise le terminal courant (équivalent de `cd dir && exec bin`).
func runInTerminal(bin, dir string) error {
	path, err := exec.LookPath(bin)
	if err != nil {
		ensurePATH()
		path, err = exec.LookPath(bin)
	}
	if err != nil {
		return fmt.Errorf("%s introuvable", bin)
	}
	if err := os.Chdir(dir); err != nil {
		return err
	}
	env := os.Environ()
	if bin == "claude" {
		env = append(env, "CLAUDE_CODE_FORCE_FULL_LOGO=1")
	}
	return syscall.Exec(path, []string{bin}, env)
}

// openVSCode ouvre le workspace dans VSCode (détaché), sans fermer le terminal.
func openVSCode(dir string) error {
	if path, err := exec.LookPath("code"); err == nil {
		cmd := exec.Command(path, dir)
		if err := cmd.Start(); err != nil {
			return err
		}
		_ = cmd.Process.Release()
	} else {
		if err := exec.Command("open", "-a", "Visual Studio Code", dir).Run(); err != nil {
			return fmt.Errorf("VSCode introuvable")
		}
	}
	return nil
}

// openFinder ouvre un dossier dans le Finder (sans quitter bagent).
func openFinder(path string) error {
	return exec.Command("open", path).Start()
}
