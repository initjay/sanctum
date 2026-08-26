package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/initjay/sanctum/internal/profile"
)

func runAdd(t *testing.T, getDeps depsFunc, args []string, stdin string) (stdout, stderr string, err error) {
	t.Helper()

	root := newRootCmdWithDeps(getDeps)
	var out, errOut bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errOut)
	root.SetIn(strings.NewReader(stdin))
	root.SetArgs(append([]string{"profile", "add"}, args...))

	err = root.Execute()
	return out.String(), errOut.String(), err
}

func TestProfileAddNonInteractiveAPIKeyFromStdin(t *testing.T) {
	getDeps := fakeDeps(t, nil, nil)
	configDir := filepath.Join(t.TempDir(), "work-acme")

	_, _, err := runAdd(t, getDeps, []string{
		"work-acme",
		"--non-interactive",
		"--credential-type", "api-key",
		"--api-key-stdin",
		"--config-dir", configDir,
		"--label", "ACME Corp",
	}, "sk-ant-abcdefghijklmnop1234\n")

	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	d, err := getDeps()
	if err != nil {
		t.Fatalf("getDeps: %v", err)
	}

	p, err := d.profiles.Get("work-acme")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if p.Label != "ACME Corp" || p.CredentialType != profile.CredentialAPIKey || p.ConfigDir != configDir {
		t.Errorf("unexpected profile: %+v", p)
	}

	value, err := d.secrets.Get("work-acme")
	if err != nil {
		t.Fatalf("secrets.Get: %v", err)
	}
	if value != "sk-ant-abcdefghijklmnop1234" {
		t.Errorf("got secret %q, want sk-ant-abcdefghijklmnop1234", value)
	}
}

func TestProfileAddCreatesConfigDir(t *testing.T) {
	getDeps := fakeDeps(t, nil, nil)
	configDir := filepath.Join(t.TempDir(), "nested", "work-acme")

	_, _, err := runAdd(t, getDeps, []string{
		"work-acme", "--non-interactive", "--credential-type", "api-key",
		"--api-key-stdin", "--config-dir", configDir,
	}, "sk-secret-value-1234567\n")

	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	info, statErr := os.Stat(configDir)
	if statErr != nil {
		t.Fatalf("expected config dir to be created, stat failed: %v", statErr)
	}
	if !info.IsDir() {
		t.Errorf("expected %s to be a directory", configDir)
	}
}

func TestProfileAddRejectsDuplicateName(t *testing.T) {
	getDeps := fakeDeps(t, []profile.Profile{
		{Name: "work-acme", CredentialType: profile.CredentialAPIKey, ConfigDir: t.TempDir()},
	}, map[string]string{"work-acme": "sk-existing"})

	_, _, err := runAdd(t, getDeps, []string{
		"work-acme", "--non-interactive", "--credential-type", "api-key",
		"--api-key-stdin",
	}, "sk-new-value-1234567890\n")

	if err == nil {
		t.Fatalf("expected an error for a duplicate profile name")
	}
}

func TestProfileAddRejectsInvalidName(t *testing.T) {
	getDeps := fakeDeps(t, nil, nil)

	_, _, err := runAdd(t, getDeps, []string{
		"bad name with spaces", "--non-interactive", "--credential-type", "api-key",
		"--api-key-stdin",
	}, "sk-value-1234567890\n")

	if err == nil {
		t.Fatalf("expected an error for an invalid profile name")
	}
}

func TestProfileAddNonInteractiveRequiresCredentialType(t *testing.T) {
	getDeps := fakeDeps(t, nil, nil)

	_, _, err := runAdd(t, getDeps, []string{"work-acme", "--non-interactive"}, "")

	if err == nil {
		t.Fatalf("expected an error requiring --credential-type")
	}
}

func TestProfileAddNonInteractiveRejectsOAuth(t *testing.T) {
	getDeps := fakeDeps(t, nil, nil)

	_, _, err := runAdd(t, getDeps, []string{
		"work-acme", "--non-interactive", "--credential-type", "oauth",
	}, "")

	if err == nil {
		t.Fatalf("expected --non-interactive with oauth to be rejected before ever touching the claude binary")
	}
}

func TestProfileAddUnknownCredentialTypeFlag(t *testing.T) {
	getDeps := fakeDeps(t, nil, nil)

	_, _, err := runAdd(t, getDeps, []string{
		"work-acme", "--non-interactive", "--credential-type", "bogus",
	}, "")

	if err == nil {
		t.Fatalf("expected an error for an unrecognized --credential-type value")
	}
}

func TestProfileAddRejectsEmptySecret(t *testing.T) {
	getDeps := fakeDeps(t, nil, nil)

	_, _, err := runAdd(t, getDeps, []string{
		"work-acme", "--non-interactive", "--credential-type", "api-key", "--api-key-stdin",
		"--config-dir", filepath.Join(t.TempDir(), "work-acme"),
	}, "\n")

	if err == nil {
		t.Fatalf("expected an error for an empty secret")
	}

	if _, err := (func() (profile.Profile, error) {
		d, derr := getDeps()
		if derr != nil {
			return profile.Profile{}, derr
		}
		return d.profiles.Get("work-acme")
	})(); err == nil {
		t.Fatalf("expected no profile to have been created after an empty secret was rejected")
	}
}

func TestProfileAddNonEmptyConfigDirRequiresReuseFlagNonInteractively(t *testing.T) {
	getDeps := fakeDeps(t, nil, nil)
	configDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(configDir, "settings.json"), []byte("{}"), 0o600); err != nil {
		t.Fatalf("seeding existing config dir: %v", err)
	}

	_, _, err := runAdd(t, getDeps, []string{
		"work-acme", "--non-interactive", "--credential-type", "api-key",
		"--api-key-stdin", "--config-dir", configDir,
	}, "sk-value-1234567890\n")

	if err == nil {
		t.Fatalf("expected an error for a non-empty config dir without --reuse-config-dir")
	}
}

func TestProfileAddReuseConfigDirFlagAllowsNonEmptyDir(t *testing.T) {
	getDeps := fakeDeps(t, nil, nil)
	configDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(configDir, "settings.json"), []byte("{}"), 0o600); err != nil {
		t.Fatalf("seeding existing config dir: %v", err)
	}

	_, _, err := runAdd(t, getDeps, []string{
		"work-acme", "--non-interactive", "--credential-type", "api-key",
		"--api-key-stdin", "--config-dir", configDir, "--reuse-config-dir",
	}, "sk-value-1234567890\n")

	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
}

func TestProfileAddRollsBackProfileWhenSecretSaveFails(t *testing.T) {
	store := newProfileTestStore(t)
	broken := brokenSecretStore{err: errWriteFailed}
	getDeps := func() (deps, error) {
		return deps{profiles: store, secrets: broken}, nil
	}

	_, _, err := runAdd(t, getDeps, []string{
		"work-acme", "--non-interactive", "--credential-type", "api-key",
		"--api-key-stdin", "--config-dir", t.TempDir(),
	}, "sk-value-1234567890\n")

	if err == nil {
		t.Fatalf("expected an error when the secret save fails")
	}

	if _, err := store.Get("work-acme"); err != profile.ErrNotFound {
		t.Fatalf("expected the profile to be rolled back, got err %v", err)
	}
}

func TestProfileAddInteractivePromptsForEverything(t *testing.T) {
	getDeps := fakeDeps(t, nil, nil)
	configDir := filepath.Join(t.TempDir(), "personal")

	// --config-dir is passed as a flag below, so resolveConfigDir never
	// prompts for it, only label, credential type choice, the secret
	// value, base url, and default model are prompted interactively here.
	stdin := strings.Join([]string{
		"My personal account",
		"1",
		"sk-personal-1234567890",
		"",
		"",
	}, "\n") + "\n"

	stdout, _, err := runAdd(t, getDeps, []string{"personal", "--config-dir", configDir}, stdin)
	if err != nil {
		t.Fatalf("Execute: %v\nstdout was:\n%s", err, stdout)
	}

	d, err := getDeps()
	if err != nil {
		t.Fatalf("getDeps: %v", err)
	}
	p, err := d.profiles.Get("personal")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if p.Label != "My personal account" || p.CredentialType != profile.CredentialAPIKey {
		t.Errorf("unexpected profile: %+v", p)
	}
}

// TestProfileAddStdinEOFDoesNotSpinForever reproduces a bug found during
// review: if stdin ran out before the credential type prompt got a valid
// "1" or "2" answer, the retry loop treated end of input the same as an
// empty answer and looped forever without ever blocking, pinning a CPU
// core. Running this off the main goroutine with a timeout is what makes
// a regression here fail the test instead of hanging it.
func TestProfileAddStdinEOFDoesNotSpinForever(t *testing.T) {
	getDeps := fakeDeps(t, nil, nil)

	done := make(chan error, 1)
	go func() {
		_, _, err := runAdd(t, getDeps, []string{"spintest", "--config-dir", t.TempDir()}, "somelabel\n")
		done <- err
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatalf("expected an error when stdin runs out before a credential type is chosen")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("profile add spun forever instead of erroring out on stdin EOF")
	}
}

func TestProfileAddRejectsConfigDirAlreadyUsedByAnotherProfile(t *testing.T) {
	sharedDir := t.TempDir()
	getDeps := fakeDeps(t, []profile.Profile{
		{Name: "existing", CredentialType: profile.CredentialAPIKey, ConfigDir: sharedDir},
	}, map[string]string{"existing": "sk-existing-value-1234567890"})

	_, _, err := runAdd(t, getDeps, []string{
		"new-profile", "--non-interactive", "--credential-type", "api-key",
		"--api-key-stdin", "--config-dir", sharedDir, "--reuse-config-dir",
	}, "sk-new-value-1234567890\n")

	if err == nil {
		t.Fatalf("expected an error when a config dir is already claimed by another profile")
	}
}

func TestProfileAddTightensPermissionsOnReusedConfigDir(t *testing.T) {
	getDeps := fakeDeps(t, nil, nil)
	configDir := t.TempDir()
	if err := os.Chmod(configDir, 0o755); err != nil {
		t.Fatalf("chmod setup: %v", err)
	}

	_, _, err := runAdd(t, getDeps, []string{
		"work-acme", "--non-interactive", "--credential-type", "api-key",
		"--api-key-stdin", "--config-dir", configDir,
	}, "sk-value-1234567890\n")

	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	info, statErr := os.Stat(configDir)
	if statErr != nil {
		t.Fatalf("Stat: %v", statErr)
	}
	if info.Mode().Perm() != 0o700 {
		t.Errorf("expected config dir permissions 0700, got %v", info.Mode().Perm())
	}
}

// TestProfileAddSecretPromptDoesNotHangWhenStdinIsSubstituted guards the
// fix for a real hang found during review: term.IsTerminal/term.ReadPassword
// used to check the process's real os.Stdin directly, so a test run under
// an actual pty would block waiting for terminal input the test's fake
// stdin could never provide, ignoring root.SetIn entirely.
// isRealTerminalStdin fixes this by requiring cmd.InOrStdin() to literally
// be os.Stdin before ever treating stdin as a real terminal, which is
// never true here since SetIn always substitutes it. This deliberately
// does not pass --api-key-stdin, so readSecretValue actually evaluates
// isRealTerminalStdin instead of short circuiting past it.
func TestProfileAddSecretPromptDoesNotHangWhenStdinIsSubstituted(t *testing.T) {
	getDeps := fakeDeps(t, nil, nil)

	done := make(chan error, 1)
	go func() {
		_, _, err := runAdd(t, getDeps, []string{
			"work-acme", "--non-interactive", "--credential-type", "api-key",
			"--config-dir", t.TempDir(),
		}, "sk-value-1234567890\n")
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Execute: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("profile add hung instead of using the fake stdin")
	}
}
