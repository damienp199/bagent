package app

import "testing"

func TestRemapPath(t *testing.T) {
	cases := []struct{ line, old, np, want string }{
		{"/a/old", "/a/old", "/a/new", "/a/new"},         // exact
		{">/a/old", "/a/old", "/a/new", ">/a/new"},       // groupe
		{"/a/old/sub", "/a/old", "/a/new", "/a/new/sub"}, // préfixe
		{"/a/other", "/a/old", "/a/new", "/a/other"},     // inchangé
	}
	for _, c := range cases {
		if got := remapPath(c.line, c.old, c.np); got != c.want {
			t.Errorf("remapPath(%q) = %q, attendu %q", c.line, got, c.want)
		}
	}
}

func TestRenameDir(t *testing.T) {
	parent := t.TempDir()
	old, _ := createDir(parent, "ancien")
	np, err := renameDir(old, "nouveau")
	if err != nil {
		t.Fatalf("renameDir: %v", err)
	}
	if !isDir(np) || isDir(old) {
		t.Fatalf("renommage incomplet: old=%v new=%v", isDir(old), isDir(np))
	}
}
