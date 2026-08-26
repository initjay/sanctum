package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/initjay/sanctum/internal/profile"
)

func TestProfileListTableOutput(t *testing.T) {
	getDeps := fakeDeps(t, []profile.Profile{
		{Name: "work-acme", Label: "ACME", CredentialType: profile.CredentialAPIKey, ConfigDir: "/home/work-acme"},
		{Name: "personal", Label: "Personal", CredentialType: profile.CredentialOAuthToken, ConfigDir: "/home/personal"},
	}, map[string]string{
		"work-acme": "sk-ant-abcdefghijklmnop1234",
		"personal":  "oauth-token-value-5678",
	})

	root := newRootCmdWithDeps(getDeps)
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetArgs([]string{"profile", "list"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "work-acme") || !strings.Contains(out, "personal") {
		t.Fatalf("expected both profile names in output, got:\n%s", out)
	}
	if !strings.Contains(out, "1234") {
		t.Errorf("expected masked secret suffix in output, got:\n%s", out)
	}
	if strings.Contains(out, "sk-ant-abcdefghijklmnop1234") || strings.Contains(out, "oauth-token-value-5678") {
		t.Fatalf("raw secret leaked into list output:\n%s", out)
	}
}

func TestProfileListEmptyStore(t *testing.T) {
	getDeps := fakeDeps(t, nil, nil)

	root := newRootCmdWithDeps(getDeps)
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetArgs([]string{"profile", "list"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if !strings.Contains(stdout.String(), "no profiles yet") {
		t.Errorf("expected an empty state message, got:\n%s", stdout.String())
	}
}

func TestProfileListMissingSecretShowsPlaceholder(t *testing.T) {
	getDeps := fakeDeps(t, []profile.Profile{
		{Name: "orphaned", CredentialType: profile.CredentialAPIKey, ConfigDir: "/home/orphaned"},
	}, nil)

	root := newRootCmdWithDeps(getDeps)
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetArgs([]string{"profile", "list"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if !strings.Contains(stdout.String(), "no secret found") {
		t.Errorf("expected a missing secret placeholder, got:\n%s", stdout.String())
	}
}

func TestProfileListJSONOutput(t *testing.T) {
	getDeps := fakeDeps(t, []profile.Profile{
		{Name: "work-acme", Label: "ACME", CredentialType: profile.CredentialAPIKey, ConfigDir: "/home/work-acme"},
	}, map[string]string{"work-acme": "sk-ant-abcdefghijklmnop1234"})

	root := newRootCmdWithDeps(getDeps)
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetArgs([]string{"profile", "list", "--json"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	var entries []profileListEntry
	if err := json.Unmarshal(stdout.Bytes(), &entries); err != nil {
		t.Fatalf("unmarshaling JSON output: %v\noutput was:\n%s", err, stdout.String())
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Name != "work-acme" {
		t.Errorf("expected name work-acme, got %q", entries[0].Name)
	}
	if entries[0].MaskedSecret != "******1234" {
		t.Errorf("expected masked secret, got %q", entries[0].MaskedSecret)
	}
	if strings.Contains(stdout.String(), "sk-ant-abcdefghijklmnop1234") {
		t.Fatalf("raw secret leaked into JSON output")
	}
}
