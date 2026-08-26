package profile

import (
	"fmt"

	"github.com/initjay/sanctum/internal/secret"
)

// ResolvedEnv is everything needed to run a Claude Code session scoped to
// one profile: which env vars to set, which to explicitly unset, and where
// the profile's isolated Claude config lives.
type ResolvedEnv struct {
	ProfileName string
	ConfigDir   string
	Vars        map[string]string
	UnsetVars   []string
}

// ResolveProfile loads a profile and its secret and computes the full
// environment a session for it should run with. It has no CLI-specific
// side effects, no stdout writes, no os.Exit, so it can be called from a
// CLI command, a test, or eventually a local HTTP handler behind a future
// GUI, unchanged.
func ResolveProfile(store *Store, secrets secret.Store, name string) (ResolvedEnv, error) {
	p, err := store.Get(name)
	if err != nil {
		return ResolvedEnv{}, err
	}

	// Store.Add/Update reject both of these today, but a hand edited
	// profiles.json or a profile written by some future, older, or buggy
	// version of sanctum could still land invalid data on disk. Refusing to
	// resolve it is the only safe move: an invalid CredentialType would
	// leave both possible credential env vars untouched, and an empty
	// ConfigDir would resolve to CLAUDE_CONFIG_DIR="", letting Claude Code
	// silently fall back to its default ~/.claude, either of which
	// defeats isolation instead of failing loudly.
	if !p.CredentialType.Valid() {
		return ResolvedEnv{}, fmt.Errorf("profile %q has an invalid credential type %q", p.Name, p.CredentialType)
	}
	if p.ConfigDir == "" {
		return ResolvedEnv{}, fmt.Errorf("profile %q has no config dir set", p.Name)
	}

	value, err := secrets.Get(name)
	if err != nil {
		return ResolvedEnv{}, err
	}

	vars := BuildEnv(p, value)
	unsetVars := BuildUnsetList(p)

	// BuildEnv and BuildUnsetList are meant to be disjoint: nothing should
	// ever appear in both. That's what lets sanctum env and sanctum shell
	// agree on the result even though they apply Vars/UnsetVars in
	// different orders internally (env prints exports then unsets, shell
	// drops-then-overlays). If that invariant ever broke, the two commands
	// would silently produce different environments for the same profile.
	// Refusing to resolve at all is safer than letting them quietly
	// diverge.
	for _, unsetName := range unsetVars {
		if _, ok := vars[unsetName]; ok {
			return ResolvedEnv{}, fmt.Errorf("internal error: profile %q would both set and unset %s", p.Name, unsetName)
		}
	}

	return ResolvedEnv{
		ProfileName: p.Name,
		ConfigDir:   p.ConfigDir,
		Vars:        vars,
		UnsetVars:   unsetVars,
	}, nil
}
