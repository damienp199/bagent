package app

import "testing"

func TestParseOpenArgs(t *testing.T) {
	cases := []struct {
		name       string
		args       []string
		wantAction string
		wantTarget string
	}{
		{"tool code", []string{"--tool=code", "/a/b"}, "vscode", "/a/b"},
		{"tool vscode", []string{"--tool=vscode", "/a/b"}, "vscode", "/a/b"},
		{"tool claude", []string{"--tool=claude", "/a/b"}, "claude", "/a/b"},
		{"tool codex", []string{"--tool=codex", "/a/b"}, "codex", "/a/b"},
		{"defaut code", []string{"/a/b"}, "vscode", "/a/b"},
		{"ordre inverse", []string{"/a/b", "--tool=claude"}, "claude", "/a/b"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			action, target := parseOpenArgs(c.args)
			if action != c.wantAction {
				t.Errorf("action = %q, attendu %q", action, c.wantAction)
			}
			if target != c.wantTarget {
				t.Errorf("target = %q, attendu %q", target, c.wantTarget)
			}
		})
	}
}

func TestParseOpenArgsNoTarget(t *testing.T) {
	if _, target := parseOpenArgs([]string{"--tool=code"}); target != "" {
		t.Errorf("target = %q, attendu vide", target)
	}
}
