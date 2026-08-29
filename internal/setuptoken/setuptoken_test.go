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

// TestParseTokenDoesNotMistakeALoginURLForTheToken reproduces a false
// positive found during review: a line whose label happens to contain the
// word "token" but whose value is actually a URL (a very plausible real
// line from an OAuth login flow) must not be returned as the token.
func TestParseTokenDoesNotMistakeALoginURLForTheToken(t *testing.T) {
	output := "Login token URL: https://console.anthropic.com/oauth/authorize?scope=abcdefghijklmnop\n"

	if got := parseToken(output); got != "" {
		t.Errorf("expected the URL to be rejected, got %q", got)
	}
}

func TestParseTokenStillFindsRealTokenAfterAURLLine(t *testing.T) {
	output := "Login token URL: https://console.anthropic.com/oauth/authorize?scope=abcdefghijklmnop\n" +
		"Token: abcdefghijklmnopqrstuvwxyz\n"

	if got := parseToken(output); got != "abcdefghijklmnopqrstuvwxyz" {
		t.Errorf("got %q, want abcdefghijklmnopqrstuvwxyz", got)
	}
}

func TestParseTokenRequiresTokenAsAWholeWordInTheLabel(t *testing.T) {
	// "tokenizer" contains "token" as a substring but isn't the word
	// "token", this must not match.
	output := "Tokenizer version: abcdefghijklmnopqrstuvwxyz\n"

	if got := parseToken(output); got != "" {
		t.Errorf("expected no match for a label that only contains token as a substring, got %q", got)
	}
}

// TestParseTokenMatchesRealOutputShape reproduces the actual shape a real
// `claude setup-token` run produces: a label line ending in a bare colon,
// a blank line, then the token alone on its own line. This is the real
// format that the original same-line-only heuristic missed. The token
// value here is fabricated, shaped like a real one (sk-ant-oat prefix,
// version, base64url body) but not an actual credential.
func TestParseTokenMatchesRealOutputShape(t *testing.T) {
	output := "Your OAuth token (valid for 1 year):\n\n" +
		"sk-ant-oat01-fakeTokenValueForTestingPurposesOnly1234567890ABCD\n"

	want := "sk-ant-oat01-fakeTokenValueForTestingPurposesOnly1234567890ABCD"
	if got := parseToken(output); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestParseTokenPrefixMatchIgnoresLabelWording confirms the confirmed
// prefix pattern is matched directly regardless of what surrounds it,
// so it doesn't depend on guessing label wording at all.
func TestParseTokenPrefixMatchIgnoresLabelWording(t *testing.T) {
	output := "some completely unrelated wording with no mention of the word at all:\n\n" +
		"sk-ant-oat01-fakeTokenValueForTestingPurposesOnly1234567890ABCD\n"

	want := "sk-ant-oat01-fakeTokenValueForTestingPurposesOnly1234567890ABCD"
	if got := parseToken(output); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestParseTokenFallbackHandlesValueOnNextLine exercises the label based
// fallback's own next-line handling directly, using a value that doesn't
// match the confirmed prefix pattern, so this only passes if the fallback
// logic itself (not the regex) correctly walks past the blank line to find
// the value. This is what keeps the parser working if some future claude
// version changes the token's prefix again.
func TestParseTokenFallbackHandlesValueOnNextLine(t *testing.T) {
	output := "Access token:\n\nabcdefghijklmnopqrstuvwxyz\n"

	if got := parseToken(output); got != "abcdefghijklmnopqrstuvwxyz" {
		t.Errorf("got %q, want abcdefghijklmnopqrstuvwxyz", got)
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
