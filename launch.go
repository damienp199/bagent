package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// toolAvailable indique si un exécutable est présent dans le PATH.
func toolAvailable(bin string) bool {
	_, err := exec.LookPath(bin)
	return err == nil
}

// runInTerminal remplace le process courant par la commande, dans le workspace.
// Réutilise le terminal courant (équivalent de `cd dir && exec bin`).
func runInTerminal(bin, dir string) error {
	path, err := exec.LookPath(bin)
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

// openVSCode ouvre le workspace dans VSCode (détaché) puis ferme le terminal.
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
	closeTerminal()
	return nil
}

// currentTTY renvoie le périphérique tty du terminal courant.
func currentTTY() string {
	cmd := exec.Command("tty")
	cmd.Stdin = os.Stdin
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// closeTerminal ferme la fenêtre du terminal courant (Terminal.app / iTerm2).
func closeTerminal() {
	tty := currentTTY()
	if tty == "" {
		return
	}
	var script string
	switch os.Getenv("TERM_PROGRAM") {
	case "Apple_Terminal":
		script = fmt.Sprintf(`tell application "Terminal"
  repeat with w in windows
    repeat with t in tabs of w
      if tty of t is "%s" then
        close w
        return
      end if
    end repeat
  end repeat
end tell`, tty)
	case "iTerm.app":
		script = fmt.Sprintf(`tell application "iTerm2"
  repeat with w in windows
    repeat with t in tabs of w
      repeat with s in sessions of t
        if tty of s is "%s" then
          close w
        end if
      end repeat
    end repeat
  end repeat
end tell`, tty)
	default:
		return
	}
	_ = exec.Command("osascript", "-e", script).Run()
}
