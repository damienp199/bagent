package app

import (
	"os"
	"testing"
)

// Simule un lancement hors shell interactif (PATH sans ~/.local/bin) :
// toolAvailable doit récupérer le PATH complet via le shell de login.
func TestToolAvailableEnrichesPATH(t *testing.T) {
	old := os.Getenv("PATH")
	defer os.Setenv("PATH", old)
	pathEnriched = false
	os.Setenv("PATH", "/usr/bin:/bin")

	if !toolAvailable("claude") {
		t.Errorf("claude devrait être trouvé après enrichissement du PATH")
	}
	if !toolAvailable("codex") {
		t.Errorf("codex devrait être trouvé après enrichissement du PATH")
	}
}
