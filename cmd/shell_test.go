package cmd

import (
	"bytes"
	"testing"
)

// The happy path of `sanctum shell <profile>` ends in syscall.Exec, which
// replaces the test process itself, so it can't be exercised in process
// here. It's covered by the manual verification checklist instead. What
// can and does get tested here is everything before that point: shell
// path resolution, and that an unresolvable profile returns an error
// instead of ever reaching Exec.

func TestResolveShellPathPrefersOverride(t *testing.T) {
	t.Setenv("SHELL", "/bin/bash")

	if got := resolveShellPath("/bin/fish"); got != "/bin/fish" {
		t.Errorf("got %q, want /bin/fish", got)
	}
}

func TestResolveShellPathFallsBackToEnv(t *testing.T) {
	t.Setenv("SHELL", "/bin/bash")

	if got := resolveShellPath(""); got != "/bin/bash" {
		t.Errorf("got %q, want /bin/bash", got)
	}
}

func TestResolveShellPathFallsBackToDefault(t *testing.T) {
	t.Setenv("SHELL", "")

	if got := resolveShellPath(""); got != "/bin/zsh" {
		t.Errorf("got %q, want /bin/zsh", got)
	}
}

func TestShellCmdUnknownProfileReturnsErrorBeforeExec(t *testing.T) {
	getDeps := fakeDeps(t, nil, nil)

	root := newRootCmdWithDeps(getDeps)
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"shell", "ghost"})

	// If this ever reached execshell.Exec, the test process itself would
	// be replaced and this test would never report a result. Returning
	// here at all is part of what's being verified.
	if err := root.Execute(); err == nil {
		t.Fatalf("expected an error for an unknown profile")
	}
}
