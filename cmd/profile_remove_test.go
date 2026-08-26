package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/initjay/sanctum/internal/profile"
)

func runRemove(t *testing.T, getDeps depsFunc, args []string, stdin string) (stdout string, err error) {
	t.Helper()

	root := newRootCmdWithDeps(getDeps)
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetIn(strings.NewReader(stdin))
	root.SetArgs(append([]string{"profile", "remove"}, args...))

	err = root.Execute()
	return out.String(), err
}

func TestProfileRemoveWithYesFlagSkipsPrompt(t *testing.T) {
	getDeps := fakeDeps(t, []profile.Profile{
		{Name: "work-acme", CredentialType: profile.CredentialAPIKey, ConfigDir: t.TempDir()},
	}, map[string]string{"work-acme": "sk-value-1234567890"})

	_, err := runRemove(t, getDeps, []string{"work-acme", "--yes"}, "")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	d, _ := getDeps()
	if _, err := d.profiles.Get("work-acme"); err != profile.ErrNotFound {
		t.Fatalf("expected the profile to be removed, got err %v", err)
	}
	if _, err := d.secrets.Get("work-acme"); err == nil {
		t.Fatalf("expected the secret to be removed")
	}
}

func TestProfileRemoveInteractiveConfirm(t *testing.T) {
	getDeps := fakeDeps(t, []profile.Profile{
		{Name: "work-acme", CredentialType: profile.CredentialAPIKey, ConfigDir: t.TempDir()},
	}, map[string]string{"work-acme": "sk-value-1234567890"})

	_, err := runRemove(t, getDeps, []string{"work-acme"}, "y\n")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	d, _ := getDeps()
	if _, err := d.profiles.Get("work-acme"); err != profile.ErrNotFound {
		t.Fatalf("expected the profile to be removed, got err %v", err)
	}
}

func TestProfileRemoveInteractiveDecline(t *testing.T) {
	getDeps := fakeDeps(t, []profile.Profile{
		{Name: "work-acme", CredentialType: profile.CredentialAPIKey, ConfigDir: t.TempDir()},
	}, map[string]string{"work-acme": "sk-value-1234567890"})

	_, err := runRemove(t, getDeps, []string{"work-acme"}, "n\n")
	if err == nil {
		t.Fatalf("expected an error when declining the confirmation")
	}

	d, _ := getDeps()
	if _, err := d.profiles.Get("work-acme"); err != nil {
		t.Fatalf("expected the profile to survive a declined removal, got err %v", err)
	}
}

func TestProfileRemoveDoesNotDeleteConfigDir(t *testing.T) {
	configDir := t.TempDir()
	marker := filepath.Join(configDir, "settings.json")
	if err := os.WriteFile(marker, []byte("{}"), 0o600); err != nil {
		t.Fatalf("seeding config dir: %v", err)
	}

	getDeps := fakeDeps(t, []profile.Profile{
		{Name: "work-acme", CredentialType: profile.CredentialAPIKey, ConfigDir: configDir},
	}, map[string]string{"work-acme": "sk-value-1234567890"})

	stdout, err := runRemove(t, getDeps, []string{"work-acme", "--yes"}, "")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if !strings.Contains(stdout, configDir) {
		t.Errorf("expected the leftover config dir path to be printed, got:\n%s", stdout)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("expected the config dir to be left in place, stat failed: %v", err)
	}
}

func TestProfileRemoveUnknownProfileReturnsError(t *testing.T) {
	getDeps := fakeDeps(t, nil, nil)

	_, err := runRemove(t, getDeps, []string{"ghost", "--yes"}, "")

	if err == nil {
		t.Fatalf("expected an error for an unknown profile")
	}
}

func TestProfileRemoveStillRemovesMetadataButFailsWhenSecretDeleteFails(t *testing.T) {
	store := newProfileTestStore(t)
	if err := store.Add(profile.Profile{
		Name:           "work-acme",
		CredentialType: profile.CredentialAPIKey,
		ConfigDir:      t.TempDir(),
	}); err != nil {
		t.Fatalf("seeding profile: %v", err)
	}

	broken := brokenSecretStore{err: errWriteFailed}
	getDeps := func() (deps, error) {
		return deps{profiles: store, secrets: broken}, nil
	}

	root := newRootCmdWithDeps(getDeps)
	var out, errOut bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errOut)
	root.SetArgs([]string{"profile", "remove", "work-acme", "--yes"})

	// A real keychain delete failure (not just "already missing", which
	// secret.Store.Delete already treats as success on its own) has to
	// make the command exit non-zero, so a script checking the exit code
	// can tell the secret might still be sitting in the keychain, even
	// though the profile metadata is still removed either way so it can
	// never turn into an unremovable ghost profile.
	if err := root.Execute(); err == nil {
		t.Fatalf("expected a non-zero exit when the keychain delete genuinely fails")
	}

	if !strings.Contains(out.String(), "Removed profile") {
		t.Errorf("expected the profile removal to still be reported as done, got stdout %q", out.String())
	}

	if _, err := store.Get("work-acme"); err != profile.ErrNotFound {
		t.Fatalf("expected the profile metadata to still be removed despite the keychain error, got %v", err)
	}
}
