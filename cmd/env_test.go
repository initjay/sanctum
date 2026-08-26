package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/initjay/sanctum/internal/profile"
)

func TestEnvCmdPrintsExportsAndUnsets(t *testing.T) {
	getDeps := fakeDeps(t, []profile.Profile{
		{
			Name:           "work-acme",
			CredentialType: profile.CredentialAPIKey,
			ConfigDir:      "/home/work-acme",
		},
	}, map[string]string{"work-acme": "sk-secret"})

	root := newRootCmdWithDeps(getDeps)
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"env", "work-acme"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "export ANTHROPIC_API_KEY='sk-secret'") {
		t.Errorf("expected ANTHROPIC_API_KEY export in output, got:\n%s", out)
	}
	if !strings.Contains(out, "export CLAUDE_CONFIG_DIR='/home/work-acme'") {
		t.Errorf("expected CLAUDE_CONFIG_DIR export in output, got:\n%s", out)
	}
	if !strings.Contains(out, "unset CLAUDE_CODE_OAUTH_TOKEN") {
		t.Errorf("expected CLAUDE_CODE_OAUTH_TOKEN unset in output, got:\n%s", out)
	}
	if stderr.Len() != 0 {
		t.Errorf("expected nothing on stderr, got %q", stderr.String())
	}
}

func TestEnvCmdUnknownProfileReturnsError(t *testing.T) {
	getDeps := fakeDeps(t, nil, nil)

	root := newRootCmdWithDeps(getDeps)
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetArgs([]string{"env", "ghost"})

	if err := root.Execute(); err == nil {
		t.Fatalf("expected an error for an unknown profile")
	}
	if stdout.Len() != 0 {
		t.Errorf("expected nothing on stdout for an error case, got %q", stdout.String())
	}
}

func TestShellQuoteEscapesEmbeddedQuotes(t *testing.T) {
	got := shellQuote(`it's a secret`)
	want := `'it'\''s a secret'`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestShellQuoteEmptyString(t *testing.T) {
	if got := shellQuote(""); got != "''" {
		t.Errorf("got %q, want ''", got)
	}
}
