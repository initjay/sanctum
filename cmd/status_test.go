package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/initjay/sanctum/internal/profile"
)

func runStatus(t *testing.T, getDeps depsFunc, args []string) (stdout string, err error) {
	t.Helper()

	root := newRootCmdWithDeps(getDeps)
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs(append([]string{"status"}, args...))

	err = root.Execute()
	return out.String(), err
}

func clearCmuxEnv(t *testing.T) {
	t.Helper()
	t.Setenv("CMUX_WORKSPACE_ID", "")
	t.Setenv("CMUX_SURFACE_ID", "")
	t.Setenv("CMUX_TAB_ID", "")
}

func TestStatusNotActivated(t *testing.T) {
	clearCmuxEnv(t)
	t.Setenv("SANCTUM_PROFILE", "")
	getDeps := fakeDeps(t, nil, nil)

	_, err := runStatus(t, getDeps, nil)

	if err == nil {
		t.Fatalf("expected an error when no profile is active")
	}
}

func TestStatusActiveProfile(t *testing.T) {
	clearCmuxEnv(t)
	configDir := t.TempDir()
	t.Setenv("SANCTUM_PROFILE", "work-acme")
	t.Setenv("CLAUDE_CONFIG_DIR", configDir)

	getDeps := fakeDeps(t, []profile.Profile{
		{Name: "work-acme", Label: "ACME", CredentialType: profile.CredentialAPIKey, ConfigDir: configDir},
	}, map[string]string{"work-acme": "sk-ant-abcdefghijklmnop1234"})

	stdout, err := runStatus(t, getDeps, nil)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	for _, want := range []string{"work-acme", "ACME", "api_key", "1234"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("expected output to contain %q, got:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "warning:") {
		t.Errorf("did not expect a config dir mismatch warning, got:\n%s", stdout)
	}
	if strings.Contains(stdout, "sk-ant-abcdefghijklmnop1234") {
		t.Fatalf("raw secret leaked into status output")
	}
}

func TestStatusWarnsOnConfigDirMismatch(t *testing.T) {
	clearCmuxEnv(t)
	t.Setenv("SANCTUM_PROFILE", "work-acme")
	t.Setenv("CLAUDE_CONFIG_DIR", "/some/stale/path")

	getDeps := fakeDeps(t, []profile.Profile{
		{Name: "work-acme", CredentialType: profile.CredentialAPIKey, ConfigDir: t.TempDir()},
	}, map[string]string{"work-acme": "sk-ant-abcdefghijklmnop1234"})

	stdout, err := runStatus(t, getDeps, nil)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if !strings.Contains(stdout, "warning:") {
		t.Errorf("expected a config dir mismatch warning, got:\n%s", stdout)
	}
}

func TestStatusProfileNoLongerExists(t *testing.T) {
	clearCmuxEnv(t)
	t.Setenv("SANCTUM_PROFILE", "ghost")
	getDeps := fakeDeps(t, nil, nil)

	_, err := runStatus(t, getDeps, nil)

	if err == nil {
		t.Fatalf("expected an error when the active profile no longer exists")
	}
}

func TestStatusShowsCmuxInfo(t *testing.T) {
	t.Setenv("CMUX_WORKSPACE_ID", "workspace-123")
	t.Setenv("CMUX_SURFACE_ID", "surface-456")
	t.Setenv("CMUX_TAB_ID", "")
	configDir := t.TempDir()
	t.Setenv("SANCTUM_PROFILE", "work-acme")
	t.Setenv("CLAUDE_CONFIG_DIR", configDir)

	getDeps := fakeDeps(t, []profile.Profile{
		{Name: "work-acme", CredentialType: profile.CredentialAPIKey, ConfigDir: configDir},
	}, map[string]string{"work-acme": "sk-ant-abcdefghijklmnop1234"})

	stdout, err := runStatus(t, getDeps, nil)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if !strings.Contains(stdout, "workspace-123") || !strings.Contains(stdout, "surface-456") {
		t.Errorf("expected cmux info in output, got:\n%s", stdout)
	}
}

func TestStatusJSONOutput(t *testing.T) {
	clearCmuxEnv(t)
	configDir := t.TempDir()
	t.Setenv("SANCTUM_PROFILE", "work-acme")
	t.Setenv("CLAUDE_CONFIG_DIR", configDir)

	getDeps := fakeDeps(t, []profile.Profile{
		{Name: "work-acme", CredentialType: profile.CredentialAPIKey, ConfigDir: configDir},
	}, map[string]string{"work-acme": "sk-ant-abcdefghijklmnop1234"})

	stdout, err := runStatus(t, getDeps, []string{"--json"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	var info statusInfo
	if err := json.Unmarshal([]byte(stdout), &info); err != nil {
		t.Fatalf("unmarshaling JSON output: %v\noutput was:\n%s", err, stdout)
	}
	if !info.Active || info.ProfileName != "work-acme" {
		t.Errorf("unexpected info: %+v", info)
	}
	if !info.ConfigDirMatchesEnv {
		t.Errorf("expected config dir to match env")
	}
}
