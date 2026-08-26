package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/initjay/sanctum/internal/execshell"
	"github.com/initjay/sanctum/internal/profile"
)

// TestEnvAndShellProduceEquivalentEnvironments guards against `env` and
// `shell` drifting apart: both are meant to be thin wrappers over the same
// ResolveProfile output, so the environment sanctum shell injects into a
// spawned process and the environment `eval "$(sanctum env ...)"` produces
// in an existing shell must end up identical.
func TestEnvAndShellProduceEquivalentEnvironments(t *testing.T) {
	resolved := profile.ResolvedEnv{
		ProfileName: "work-acme",
		ConfigDir:   "/home/work-acme",
		Vars: map[string]string{
			"CLAUDE_CONFIG_DIR": "/home/work-acme",
			"ANTHROPIC_API_KEY": "sk-secret with spaces and a ' quote",
			"SANCTUM_PROFILE":   "work-acme",
		},
		UnsetVars: []string{"CLAUDE_CODE_OAUTH_TOKEN", "ANTHROPIC_AUTH_TOKEN"},
	}

	// A base environment deliberately carrying stale/adversarial values for
	// everything resolved touches, so the comparison actually proves both
	// paths override or scrub them the same way, not just that they agree
	// on an already-empty environment.
	base := []string{
		"HOME=/Users/tester",
		"CLAUDE_CODE_OAUTH_TOKEN=stale-token",
		"ANTHROPIC_API_KEY=stale-key",
		"ANTHROPIC_AUTH_TOKEN=stale-auth-token",
	}

	var buf bytes.Buffer
	if err := writeEnvScript(&buf, resolved); err != nil {
		t.Fatalf("writeEnvScript: %v", err)
	}
	fromEnvCmd := parseExportScript(t, buf.String())

	shellEnv := execshell.BuildEnv(base, resolved.Vars, resolved.UnsetVars)
	fromShellCmd := map[string]string{}
	for _, kv := range shellEnv {
		name, value, ok := strings.Cut(kv, "=")
		if ok {
			fromShellCmd[name] = value
		}
	}

	for name, want := range resolved.Vars {
		if fromEnvCmd[name] != want {
			t.Errorf("env cmd: %s = %q, want %q", name, fromEnvCmd[name], want)
		}
		if fromShellCmd[name] != want {
			t.Errorf("shell cmd: %s = %q, want %q", name, fromShellCmd[name], want)
		}
	}

	for _, name := range resolved.UnsetVars {
		if _, ok := fromShellCmd[name]; ok {
			t.Errorf("shell cmd: expected %s to be absent from the spawned environment", name)
		}
		if _, ok := fromEnvCmd[name]; ok {
			t.Errorf("env cmd: expected no export line for %s, only an unset", name)
		}
	}
}

// parseExportScript is a minimal parser for the exact output shape
// writeEnvScript produces (export NAME='value' lines using shellQuote's
// escaping), just enough to verify round trip correctness without
// depending on a real shell being available in the test environment.
func parseExportScript(t *testing.T, script string) map[string]string {
	t.Helper()
	result := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(script), "\n") {
		if !strings.HasPrefix(line, "export ") {
			continue
		}
		rest := strings.TrimPrefix(line, "export ")
		name, quoted, ok := strings.Cut(rest, "=")
		if !ok {
			t.Fatalf("malformed export line: %q", line)
		}
		result[name] = unquoteShell(t, quoted)
	}
	return result
}

func unquoteShell(t *testing.T, quoted string) string {
	t.Helper()
	if !strings.HasPrefix(quoted, "'") || !strings.HasSuffix(quoted, "'") {
		t.Fatalf("expected a single quoted value, got %q", quoted)
	}
	inner := quoted[1 : len(quoted)-1]
	return strings.ReplaceAll(inner, `'\''`, "'")
}
