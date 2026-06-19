package app

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func TestParseSemver(t *testing.T) {
	got, ok := parseSemver("v0.1.2")
	if !ok || got != [3]int{0, 1, 2} {
		t.Fatalf("parseSemver(v0.1.2) = %v,%v ; veut [0 1 2],true", got, ok)
	}
	if _, ok := parseSemver("v1.2"); ok {
		t.Fatal("parseSemver(v1.2) devrait échouer (3 segments requis)")
	}
	if _, ok := parseSemver("vX.Y.Z"); ok {
		t.Fatal("parseSemver(vX.Y.Z) devrait échouer (non numérique)")
	}
}

func TestIsNewer(t *testing.T) {
	cases := []struct {
		latest, current string
		want            bool
	}{
		{"v0.1.3", "v0.1.2", true},
		{"v0.2.0", "v0.1.9", true},
		{"v1.0.0", "v0.9.9", true},
		{"v0.1.2", "v0.1.2", false},
		{"v0.1.1", "v0.1.2", false},
		{"v0.1.3", "dev", false},
		{"pas-un-tag", "v0.1.2", false},
		{"v0.1.3", "pas-un-tag", false},
	}
	for _, c := range cases {
		if got := isNewer(c.latest, c.current); got != c.want {
			t.Errorf("isNewer(%q,%q) = %v ; veut %v", c.latest, c.current, got, c.want)
		}
	}
}

func TestLatestRelease(t *testing.T) {
	fetch := func(string) ([]byte, error) {
		return []byte(`{"tag_name":"v0.1.5","name":"v0.1.5"}`), nil
	}
	tag, err := latestRelease(fetch)
	if err != nil || tag != "v0.1.5" {
		t.Fatalf("latestRelease = %q,%v ; veut v0.1.5,nil", tag, err)
	}

	boom := func(string) ([]byte, error) { return nil, errors.New("réseau") }
	if _, err := latestRelease(boom); err == nil {
		t.Fatal("latestRelease devrait propager l'erreur de fetch")
	}

	empty := func(string) ([]byte, error) { return []byte(`{}`), nil }
	if _, err := latestRelease(empty); err == nil {
		t.Fatal("latestRelease devrait échouer sur tag_name vide")
	}
}

func TestCheckForUpdate(t *testing.T) {
	t.Setenv("HOME", t.TempDir()) // isole le cache (~/.config/bagent)
	version = "v0.1.2"
	t.Cleanup(func() { version = "dev" })

	fetch := func(string) ([]byte, error) {
		return []byte(`{"tag_name":"v0.1.5"}`), nil
	}

	// 1er appel : pas de cache → fetch, notif attendue.
	if got := checkForUpdate(1000, fetch); got != "v0.1.5" {
		t.Fatalf("checkForUpdate (frais) = %q ; veut v0.1.5", got)
	}

	// 2e appel < 24h plus tard : sert le cache même si le fetch planterait.
	boom := func(string) ([]byte, error) { return nil, errors.New("réseau") }
	if got := checkForUpdate(2000, boom); got != "v0.1.5" {
		t.Fatalf("checkForUpdate (caché) = %q ; veut v0.1.5 (cache)", got)
	}

	// > 24h plus tard : cache périmé → re-fetch.
	older := func(string) ([]byte, error) {
		return []byte(`{"tag_name":"v0.1.1"}`), nil
	}
	if got := checkForUpdate(1000+86401, older); got != "" {
		t.Fatalf("checkForUpdate (périmé, v0.1.1 <= v0.1.2) = %q ; veut \"\"", got)
	}

	// Erreur réseau sans cache valide → silencieux.
	os.RemoveAll(updateCacheFile())
	if got := checkForUpdate(99999999, boom); got != "" {
		t.Fatalf("checkForUpdate (erreur) = %q ; veut \"\"", got)
	}
}

func TestUpdateCommandString(t *testing.T) {
	got := updateShellCmd()
	if !strings.Contains(got, "install.sh") || !strings.Contains(got, "curl -fsSL") {
		t.Fatalf("updateShellCmd = %q ; doit lancer install.sh via curl", got)
	}
	if !strings.Contains(got, "https://raw.githubusercontent.com/damienp199/bagent/main/install.sh") {
		t.Fatalf("updateShellCmd = %q ; doit pointer l'URL canonique de install.sh", got)
	}
}

func TestFooterShowsUpdateNotice(t *testing.T) {
	m := model{
		pages:     []Page{{Title: "Favoris", Icon: "★", Kind: KindFavoris}},
		mode:      modeList,
		focus:     focusList,
		width:     100,
		height:    30,
		updateTag: "v0.1.9",
	}
	out := stripANSI(m.footer())
	if !strings.Contains(out, "v0.1.9") || !strings.Contains(out, "bagent --update") {
		t.Fatalf("le footer devrait annoncer la mise à jour :\n%s", out)
	}

	// Un status ponctuel a priorité et masque la notif.
	m.status = "✓ Fait"
	out = stripANSI(m.footer())
	if strings.Contains(out, "bagent --update") {
		t.Fatalf("le status doit masquer la notif :\n%s", out)
	}
}
