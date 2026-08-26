package cmd

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/initjay/sanctum/internal/profile"
)

func runEdit(t *testing.T, getDeps depsFunc, args []string, stdin string) (stdout string, err error) {
	t.Helper()

	root := newRootCmdWithDeps(getDeps)
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetIn(strings.NewReader(stdin))
	root.SetArgs(append([]string{"profile", "edit"}, args...))

	err = root.Execute()
	return out.String(), err
}

func TestProfileEditRequiresAtLeastOneFlag(t *testing.T) {
	getDeps := fakeDeps(t, []profile.Profile{
		{Name: "work-acme", CredentialType: profile.CredentialAPIKey, ConfigDir: t.TempDir()},
	}, map[string]string{"work-acme": "sk-value-1234567890"})

	_, err := runEdit(t, getDeps, []string{"work-acme"}, "")

	if err == nil {
		t.Fatalf("expected an error when no flags are given")
	}
}

func TestProfileEditUpdatesLabel(t *testing.T) {
	getDeps := fakeDeps(t, []profile.Profile{
		{Name: "work-acme", Label: "old label", CredentialType: profile.CredentialAPIKey, ConfigDir: t.TempDir()},
	}, map[string]string{"work-acme": "sk-value-1234567890"})

	_, err := runEdit(t, getDeps, []string{"work-acme", "--label", "new label"}, "")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	d, _ := getDeps()
	p, err := d.profiles.Get("work-acme")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if p.Label != "new label" {
		t.Errorf("got label %q, want new label", p.Label)
	}
}

func TestProfileEditClearsBaseURL(t *testing.T) {
	getDeps := fakeDeps(t, []profile.Profile{
		{Name: "work-acme", CredentialType: profile.CredentialAPIKey, ConfigDir: t.TempDir(), BaseURL: "https://old.example.com"},
	}, map[string]string{"work-acme": "sk-value-1234567890"})

	_, err := runEdit(t, getDeps, []string{"work-acme", "--base-url", ""}, "")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	d, _ := getDeps()
	p, err := d.profiles.Get("work-acme")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if p.BaseURL != "" {
		t.Errorf("expected base url to be cleared, got %q", p.BaseURL)
	}
}

func TestProfileEditRejectsEmptyConfigDir(t *testing.T) {
	getDeps := fakeDeps(t, []profile.Profile{
		{Name: "work-acme", CredentialType: profile.CredentialAPIKey, ConfigDir: t.TempDir()},
	}, map[string]string{"work-acme": "sk-value-1234567890"})

	_, err := runEdit(t, getDeps, []string{"work-acme", "--config-dir", ""}, "")

	if err == nil {
		t.Fatalf("expected an error when clearing config dir")
	}
}

func TestProfileEditRejectsConfigDirClaimedByAnotherProfile(t *testing.T) {
	sharedDir := t.TempDir()
	getDeps := fakeDeps(t, []profile.Profile{
		{Name: "other", CredentialType: profile.CredentialAPIKey, ConfigDir: sharedDir},
		{Name: "work-acme", CredentialType: profile.CredentialAPIKey, ConfigDir: t.TempDir()},
	}, map[string]string{
		"other":     "sk-other-value-1234567890",
		"work-acme": "sk-value-1234567890",
	})

	_, err := runEdit(t, getDeps, []string{"work-acme", "--config-dir", sharedDir}, "")

	if err == nil {
		t.Fatalf("expected an error when the new config dir is claimed by another profile")
	}
}

func TestProfileEditConfigDirDoesNotCollideWithItself(t *testing.T) {
	ownDir := t.TempDir()
	getDeps := fakeDeps(t, []profile.Profile{
		{Name: "work-acme", CredentialType: profile.CredentialAPIKey, ConfigDir: ownDir},
	}, map[string]string{"work-acme": "sk-value-1234567890"})

	// Re-pointing a profile at its own current config dir must not trip
	// the "already claimed" check against itself.
	_, err := runEdit(t, getDeps, []string{"work-acme", "--config-dir", ownDir}, "")

	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
}

func TestProfileEditRotateCredentialWithAPIKey(t *testing.T) {
	getDeps := fakeDeps(t, []profile.Profile{
		{Name: "work-acme", CredentialType: profile.CredentialAPIKey, ConfigDir: t.TempDir()},
	}, map[string]string{"work-acme": "sk-old-value-1234567890"})

	_, err := runEdit(t, getDeps, []string{"work-acme", "--rotate-credential"}, "sk-new-value-1234567890\n")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	d, _ := getDeps()
	value, err := d.secrets.Get("work-acme")
	if err != nil {
		t.Fatalf("secrets.Get: %v", err)
	}
	if value != "sk-new-value-1234567890" {
		t.Errorf("got %q, want sk-new-value-1234567890", value)
	}
}

// TestProfileEditCredentialTypeChangeForcesRotation switches FROM oauth TO
// api-key rather than the other direction deliberately: switching to
// oauth would run the real claude setup-token subprocess (through
// resolveSecret -> runSetupToken), which could actually try to launch a
// browser login flow if the claude binary happens to be on PATH, which it
// very well might be on a machine that also has Claude Code CLI installed.
// Switching to api-key exercises the same "type change forces rotation"
// code path through the safe, fully fake-able readSecretValue path
// instead.
func TestProfileEditCredentialTypeChangeForcesRotation(t *testing.T) {
	getDeps := fakeDeps(t, []profile.Profile{
		{Name: "work-acme", CredentialType: profile.CredentialOAuthToken, ConfigDir: t.TempDir()},
	}, map[string]string{"work-acme": "old-oauth-token-value-1234567890"})

	_, err := runEdit(t, getDeps, []string{
		"work-acme", "--credential-type", "api-key", "--api-key-stdin",
	}, "sk-new-value-1234567890\n")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	d, _ := getDeps()
	p, err := d.profiles.Get("work-acme")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if p.CredentialType != profile.CredentialAPIKey {
		t.Errorf("expected credential type api_key, got %q", p.CredentialType)
	}

	value, err := d.secrets.Get("work-acme")
	if err != nil {
		t.Fatalf("secrets.Get: %v", err)
	}
	if value != "sk-new-value-1234567890" {
		t.Errorf("got secret %q, want sk-new-value-1234567890", value)
	}
}

// TestProfileEditSameCredentialTypeDoesNotForceRotation confirms passing
// --credential-type with the profile's existing type is a no-op for the
// secret (only --rotate-credential or an actual type change should touch
// it), so this never risks invoking the real claude binary either.
func TestProfileEditSameCredentialTypeDoesNotForceRotation(t *testing.T) {
	getDeps := fakeDeps(t, []profile.Profile{
		{Name: "work-acme", CredentialType: profile.CredentialAPIKey, ConfigDir: t.TempDir()},
	}, map[string]string{"work-acme": "sk-original-value-1234567890"})

	_, err := runEdit(t, getDeps, []string{"work-acme", "--credential-type", "api-key"}, "")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	d, _ := getDeps()
	value, err := d.secrets.Get("work-acme")
	if err != nil {
		t.Fatalf("secrets.Get: %v", err)
	}
	if value != "sk-original-value-1234567890" {
		t.Errorf("expected the secret to be untouched, got %q", value)
	}
}

func TestProfileEditUnknownProfileReturnsError(t *testing.T) {
	getDeps := fakeDeps(t, nil, nil)

	_, err := runEdit(t, getDeps, []string{"ghost", "--label", "x"}, "")

	if err == nil {
		t.Fatalf("expected an error for an unknown profile")
	}
}

func TestProfileEditConfigDirIsAbsolute(t *testing.T) {
	getDeps := fakeDeps(t, []profile.Profile{
		{Name: "work-acme", CredentialType: profile.CredentialAPIKey, ConfigDir: t.TempDir()},
	}, map[string]string{"work-acme": "sk-value-1234567890"})

	newDir := t.TempDir()
	_, err := runEdit(t, getDeps, []string{"work-acme", "--config-dir", newDir + string(filepath.Separator)}, "")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	d, _ := getDeps()
	p, err := d.profiles.Get("work-acme")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if p.ConfigDir != newDir {
		t.Errorf("got %q, want %q (trailing slash should be normalized away)", p.ConfigDir, newDir)
	}
}
