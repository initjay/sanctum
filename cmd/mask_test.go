package cmd

import "testing"

func TestMaskSecretShortValues(t *testing.T) {
	cases := map[string]string{
		"":     "",
		"a":    "*",
		"ab":   "**",
		"abc":  "***",
		"abcd": "****",
	}

	for input, want := range cases {
		if got := maskSecret(input); got != want {
			t.Errorf("maskSecret(%q) = %q, want %q", input, got, want)
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
	secret := "sk-ant-super-secret-value-do-not-leak"
	masked := maskSecret(secret)

	if masked == secret {
		t.Fatalf("masked value equals the raw secret")
	}
	if len(masked) >= 8 && masked[:len(masked)-4] != "******" {
		t.Errorf("expected a fixed mask run, got %q", masked)
	}
}
