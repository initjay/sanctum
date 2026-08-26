package cmd

import (
	"errors"
	"fmt"

	"github.com/initjay/sanctum/internal/secret"
)

// maskSecret returns a display safe representation of a secret. Every
// result is the same fixed length regardless of input, so the output never
// leaks the secret's actual length. For secrets long enough that revealing
// a short tail is a small fraction of the whole, it appends the last 4
// characters so a user can recognize which credential they're looking at.
// Anything shorter than that is masked completely instead, since a 4
// character suffix of a short secret would give away most of it. Slicing
// happens on runes rather than bytes, so a secret containing multi-byte
// characters near the end can't produce a truncated/invalid fragment.
func maskSecret(value string) string {
	const maskRun = "******"
	const visibleChars = 4
	const minLenToReveal = 16

	runes := []rune(value)
	if len(runes) < minLenToReveal {
		return maskRun
	}

	return maskRun + string(runes[len(runes)-visibleChars:])
}

// resolveMaskedSecret looks up a profile's secret and returns a display
// string for it, distinguishing a profile that genuinely has no secret
// stored yet from a real lookup failure (a locked Keychain, a denied
// access prompt, corrupted state). Collapsing both into the same "missing"
// message would hide a real operational problem behind what looks like a
// simple setup gap.
func resolveMaskedSecret(secrets secret.Store, profileName string) string {
	value, err := secrets.Get(profileName)
	switch {
	case err == nil:
		return maskSecret(value)
	case errors.Is(err, secret.ErrNotFound):
		return "(no secret found)"
	default:
		return fmt.Sprintf("(error reading secret: %v)", err)
	}
}
