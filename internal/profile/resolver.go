package profile

import "github.com/initjay/sanctum/internal/secret"

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

	value, err := secrets.Get(name)
	if err != nil {
		return ResolvedEnv{}, err
	}

	return ResolvedEnv{
		ProfileName: p.Name,
		ConfigDir:   p.ConfigDir,
		Vars:        BuildEnv(p, value),
		UnsetVars:   BuildUnsetList(p),
	}, nil
}
