package cmd

import (
	"errors"
	"strings"
	"testing"

	"github.com/initjay/sanctum/internal/secret"
)

func TestMaskSecretShortAndEmptyValuesAreFullyMasked(t *testing.T) {
	// Anything under the reveal threshold must come back as the same fixed
	// mask, never scaled to the input's length, since that would leak how
	// long the secret is.
	for _, input := range []string{"", "a", "abcd", "short-token-12"} {
		if got := maskSecret(input); got != "******" {
			t.Errorf("maskSecret(%q) = %q, want ******", input, got)
		}
	}
}

func TestMaskSecretLongValueShowsFixedRunAndLastFourChars(t *testing.T) {
	got := maskSecret("sk-ant-abcdefghijklmnop1234")
	want := "******1234"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestMaskSecretNeverContainsTheRawValue(t *testing.T) {
	value := "sk-ant-super-secret-value-do-not-leak"
	masked := maskSecret(value)

	if masked == value {
		t.Fatalf("masked value equals the raw secret")
	}
	if strings.Contains(masked, value) {
		t.Fatalf("masked value contains the raw secret")
	}
	if !strings.HasPrefix(masked, "******") {
		t.Errorf("expected a fixed mask run prefix, got %q", masked)
	}
}

func TestMaskSecretHandlesMultiByteCharactersSafely(t *testing.T) {
	// A secret long enough to trigger the reveal path, ending in
	// multi-byte runes. Byte-index slicing here would either panic or
	// split a rune in half; rune-based slicing must not.
	value := "sk-ant-testing-🔑🔑🔑🔑"

	got := maskSecret(value)
	if !strings.HasPrefix(got, "******") {
		t.Errorf("expected a fixed mask run prefix, got %q", got)
	}
	if !strings.HasSuffix(got, "🔑🔑🔑🔑") {
		t.Errorf("expected the last 4 runes intact, got %q", got)
	}
}

func TestResolveMaskedSecretDistinguishesMissingFromRealError(t *testing.T) {
	mem := secret.NewMemStore()
	if err := mem.Set("has-secret", "sk-ant-abcdefghijklmnop1234"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	if got := resolveMaskedSecret(mem, "has-secret"); got != "******1234" {
		t.Errorf("got %q, want ******1234", got)
	}

	if got := resolveMaskedSecret(mem, "never-set"); got != "(no secret found)" {
		t.Errorf("got %q, want (no secret found)", got)
	}

	broken := brokenSecretStore{err: errors.New("keychain is locked")}
	got := resolveMaskedSecret(broken, "anything")
	if got == "(no secret found)" {
		t.Fatalf("a real store error must not be reported the same as a missing secret")
	}
	if !strings.Contains(got, "keychain is locked") {
		t.Errorf("expected the underlying error to be surfaced, got %q", got)
	}
}

// brokenSecretStore always fails with a non-ErrNotFound error, simulating
// a real Keychain problem rather than a profile with no secret stored.
type brokenSecretStore struct {
	err error
}

func (b brokenSecretStore) Get(string) (string, error) { return "", b.err }
func (b brokenSecretStore) Set(string, string) error   { return b.err }
func (b brokenSecretStore) Delete(string) error        { return b.err }
