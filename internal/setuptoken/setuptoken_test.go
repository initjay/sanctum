package setuptoken

import (
	"strings"
	"testing"
)

func TestParseTokenFindsLabeledLine(t *testing.T) {
	output := "Logging in...\nOpen this URL to continue: https://example.com\nToken: abcdefghijklmnopqrstuvwxyz\nDone.\n"

	if got := parseToken(output); got != "abcdefghijklmnopqrstuvwxyz" {
		t.Errorf("got %q, want abcdefghijklmnopqrstuvwxyz", got)
	}
}

func TestParseTokenCaseInsensitiveLabel(t *testing.T) {
	output := "OAuth TOKEN: abcdefghijklmnopqrstuvwxyz\n"

	if got := parseToken(output); got != "abcdefghijklmnopqrstuvwxyz" {
		t.Errorf("got %q, want abcdefghijklmnopqrstuvwxyz", got)
	}
}

func TestParseTokenReturnsEmptyWhenNothingMatches(t *testing.T) {
	output := "Logging in...\nSuccessfully authenticated.\n"

	if got := parseToken(output); got != "" {
		t.Errorf("expected no match, got %q", got)
	}
}

func TestParseTokenIgnoresShortOrSpacedValues(t *testing.T) {
	output := "Token: short\nAuth token: has a space in it\n"

	if got := parseToken(output); got != "" {
		t.Errorf("expected no confident match, got %q", got)
	}
}

func TestChildEnvSetsConfigDir(t *testing.T) {
	env := childEnv("/tmp/some-profile-dir")

	found := false
	for _, kv := range env {
		if kv == "CLAUDE_CONFIG_DIR=/tmp/some-profile-dir" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected CLAUDE_CONFIG_DIR to be set, got %v", env)
	}
}

func TestChildEnvExcludesCredentialVars(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "should-not-appear")
	t.Setenv("CLAUDE_CODE_OAUTH_TOKEN", "should-not-appear-either")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "nor-this-one")

	env := childEnv("/tmp/some-profile-dir")

	for _, kv := range env {
		if strings.HasPrefix(kv, "ANTHROPIC_API_KEY=") ||
			strings.HasPrefix(kv, "CLAUDE_CODE_OAUTH_TOKEN=") ||
			strings.HasPrefix(kv, "ANTHROPIC_AUTH_TOKEN=") {
			t.Errorf("expected credential vars to be excluded, found %q", kv)
		}
	}
}

func TestChildEnvKeepsPath(t *testing.T) {
	env := childEnv("/tmp/some-profile-dir")

	found := false
	for _, kv := range env {
		if strings.HasPrefix(kv, "PATH=") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected PATH to survive into the child env")
	}
}
