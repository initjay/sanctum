package cmd

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/initjay/sanctum/internal/profile"
)

func TestProfileShowTextOutput(t *testing.T) {
	configDir := t.TempDir()

	getDeps := fakeDeps(t, []profile.Profile{
		{
			Name:           "work-acme",
			Label:          "ACME Corp",
			CredentialType: profile.CredentialAPIKey,
			ConfigDir:      configDir,
			BaseURL:        "https://gateway.example.com",
		},
	}, map[string]string{"work-acme": "sk-ant-abcdefghijklmnop1234"})

	root := newRootCmdWithDeps(getDeps)
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetArgs([]string{"profile", "show", "work-acme"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	out := stdout.String()
	for _, want := range []string{"work-acme", "ACME Corp", "api_key", "1234", "gateway.example.com", "true"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected output to contain %q, got:\n%s", want, out)
		}
	}
	if strings.Contains(out, "sk-ant-abcdefghijklmnop1234") {
		t.Fatalf("raw secret leaked into show output:\n%s", out)
	}
}

func TestProfileShowConfigDirDoesNotExist(t *testing.T) {
	getDeps := fakeDeps(t, []profile.Profile{
		{
			Name:           "no-dir-yet",
			CredentialType: profile.CredentialAPIKey,
			ConfigDir:      filepath.Join(t.TempDir(), "does-not-exist"),
		},
	}, map[string]string{"no-dir-yet": "sk-secret"})

	root := newRootCmdWithDeps(getDeps)
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetArgs([]string{"profile", "show", "no-dir-yet"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if !strings.Contains(stdout.String(), "false") {
		t.Errorf("expected config dir exists to be false, got:\n%s", stdout.String())
	}
}

func TestProfileShowUnknownProfileReturnsError(t *testing.T) {
	getDeps := fakeDeps(t, nil, nil)

	root := newRootCmdWithDeps(getDeps)
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetArgs([]string{"profile", "show", "ghost"})

	if err := root.Execute(); err == nil {
		t.Fatalf("expected an error for an unknown profile")
	}
	if stdout.Len() != 0 {
		t.Errorf("expected nothing on stdout for an error case, got %q", stdout.String())
	}
}

func TestProfileShowJSONOutput(t *testing.T) {
	configDir := t.TempDir()

	getDeps := fakeDeps(t, []profile.Profile{
		{
			Name:           "work-acme",
			CredentialType: profile.CredentialOAuthToken,
			ConfigDir:      configDir,
			DefaultModel:   "claude-sonnet-5",
		},
	}, map[string]string{"work-acme": "oauth-token-value-5678"})

	root := newRootCmdWithDeps(getDeps)
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetArgs([]string{"profile", "show", "work-acme", "--json"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	var detail profileDetail
	if err := json.Unmarshal(stdout.Bytes(), &detail); err != nil {
		t.Fatalf("unmarshaling JSON output: %v\noutput was:\n%s", err, stdout.String())
	}
	if detail.Name != "work-acme" {
		t.Errorf("expected name work-acme, got %q", detail.Name)
	}
	if !detail.ConfigDirExists {
		t.Errorf("expected config dir exists to be true")
	}
	if detail.DefaultModel != "claude-sonnet-5" {
		t.Errorf("expected default model claude-sonnet-5, got %q", detail.DefaultModel)
	}
	if strings.Contains(stdout.String(), "oauth-token-value-5678") {
		t.Fatalf("raw secret leaked into JSON output")
	}
}
