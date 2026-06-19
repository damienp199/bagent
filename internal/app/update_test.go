package app

import "testing"

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
