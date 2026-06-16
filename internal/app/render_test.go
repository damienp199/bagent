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
	if m.curPage().Kind != KindProjet {
		t.Skip("aucune page de projet dans cette config")
	}
	out := stripANSI(m.View())
	if title := strings.ToUpper(m.curPage().Title); !strings.Contains(out, title) {
		t.Errorf("nom du projet en majuscules absent (%q):\n%s", title, out)
	}
	if !strings.Contains(out, "dossier") {
		t.Errorf("footer page projet sans dossier:\n%s", out)
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
