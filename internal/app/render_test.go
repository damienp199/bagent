package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTabsUppercaseAndGroupFooter(t *testing.T) {
	m := newModel()
	m.width, m.height = 100, 30
	// aller sur une page de groupe
	for i := 0; i < len(m.pages); i++ {
		if m.curPage().Kind == KindProjet {
			break
		}
		m = press(m, "right")
	}
	out := stripANSI(m.View())
	if !strings.Contains(out, "FAVORIS") {
		t.Errorf("onglets non en majuscules:\n%s", out)
	}
	if !strings.Contains(out, "dossier") {
		t.Errorf("footer page groupe sans dossier:\n%s", out)
	}
}

func TestCreateDirFlow(t *testing.T) {
	parent := t.TempDir()
	full, err := createDir(parent, "mon-projet")
	if err != nil {
		t.Fatalf("createDir: %v", err)
	}
	if !isDir(full) || filepath.Base(full) != "mon-projet" {
		t.Fatalf("dossier non créé: %s", full)
	}
	// recréer -> ErrExist
	if _, err := createDir(parent, "mon-projet"); err != os.ErrExist {
		t.Errorf("attendu ErrExist, obtenu %v", err)
	}
}

var _ = tea.KeyMsg{}
