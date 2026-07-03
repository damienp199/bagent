package app

import "testing"

func TestTerminalLaunchCmd(t *testing.T) {
	cases := []struct {
		bin, dir, want string
	}{
		{"codex", "/Users/x/AgenticOS/review", `cd '/Users/x/AgenticOS/review' && exec codex`},
		{"claude", "/Users/x/proj", `cd '/Users/x/proj' && CLAUDE_CODE_FORCE_FULL_LOGO=1 exec claude`},
		{"codex", "/Users/x/my repo", `cd '/Users/x/my repo' && exec codex`},
		{"codex", "/Users/x/o'brien", `cd '/Users/x/o'\''brien' && exec codex`},
	}
	for _, c := range cases {
		if got := terminalLaunchCmd(c.bin, c.dir); got != c.want {
			t.Errorf("terminalLaunchCmd(%q,%q) = %q, want %q", c.bin, c.dir, got, c.want)
		}
	}
}
