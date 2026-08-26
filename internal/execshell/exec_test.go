package execshell

import "testing"

func envMap(env []string) map[string]string {
	m := make(map[string]string, len(env))
	for _, kv := range env {
		for i := 0; i < len(kv); i++ {
			if kv[i] == '=' {
				m[kv[:i]] = kv[i+1:]
				break
			}
		}
	}
	return m
}

func TestBuildEnvOverlaysVars(t *testing.T) {
	base := []string{"HOME=/Users/tester", "TERM=xterm"}
	vars := map[string]string{"ANTHROPIC_API_KEY": "sk-secret"}

	got := envMap(BuildEnv(base, vars, nil))

	if got["ANTHROPIC_API_KEY"] != "sk-secret" {
		t.Errorf("expected ANTHROPIC_API_KEY to be set, got %q", got["ANTHROPIC_API_KEY"])
	}
	if got["HOME"] != "/Users/tester" {
		t.Errorf("expected HOME to survive from base, got %q", got["HOME"])
	}
}

func TestBuildEnvDropsUnsetVars(t *testing.T) {
	base := []string{"ANTHROPIC_AUTH_TOKEN=stale", "HOME=/Users/tester"}

	got := envMap(BuildEnv(base, nil, []string{"ANTHROPIC_AUTH_TOKEN"}))

	if _, ok := got["ANTHROPIC_AUTH_TOKEN"]; ok {
		t.Errorf("expected ANTHROPIC_AUTH_TOKEN to be dropped")
	}
	if got["HOME"] != "/Users/tester" {
		t.Errorf("expected HOME to survive, got %q", got["HOME"])
	}
}

func TestBuildEnvVarsOverrideStaleBaseValue(t *testing.T) {
	base := []string{"ANTHROPIC_API_KEY=stale-key"}
	vars := map[string]string{"ANTHROPIC_API_KEY": "fresh-key"}

	got := envMap(BuildEnv(base, vars, nil))

	if got["ANTHROPIC_API_KEY"] != "fresh-key" {
		t.Errorf("expected fresh-key, got %q", got["ANTHROPIC_API_KEY"])
	}

	count := 0
	for _, kv := range BuildEnv(base, vars, nil) {
		if len(kv) >= len("ANTHROPIC_API_KEY=") && kv[:len("ANTHROPIC_API_KEY=")] == "ANTHROPIC_API_KEY=" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected exactly one ANTHROPIC_API_KEY entry, got %d", count)
	}
}

func TestBuildEnvAdversarialAmbientVarsAreScrubbed(t *testing.T) {
	base := []string{
		"ANTHROPIC_API_KEY=bogus",
		"ANTHROPIC_AUTH_TOKEN=bogus",
		"AWS_PROFILE=bogus",
		"HOME=/Users/tester",
	}
	vars := map[string]string{"ANTHROPIC_API_KEY": "real-key"}
	unset := []string{"ANTHROPIC_AUTH_TOKEN", "AWS_PROFILE", "CLAUDE_CODE_OAUTH_TOKEN"}

	got := envMap(BuildEnv(base, vars, unset))

	if got["ANTHROPIC_API_KEY"] != "real-key" {
		t.Errorf("expected the real key to win, got %q", got["ANTHROPIC_API_KEY"])
	}
	for _, name := range []string{"ANTHROPIC_AUTH_TOKEN", "AWS_PROFILE", "CLAUDE_CODE_OAUTH_TOKEN"} {
		if _, ok := got[name]; ok {
			t.Errorf("expected %s to be scrubbed, but it survived", name)
		}
	}
	if got["HOME"] != "/Users/tester" {
		t.Errorf("expected unrelated vars to survive, got %q", got["HOME"])
	}
}
