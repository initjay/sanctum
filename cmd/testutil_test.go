package cmd

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/initjay/sanctum/internal/profile"
	"github.com/initjay/sanctum/internal/secret"
)

// errWriteFailed is a stand-in for a real Keychain/store failure in tests
// that need Set to fail deliberately.
var errWriteFailed = errors.New("write failed")

// newProfileTestStore returns a temp file backed profile.Store, for tests
// that need direct access to the same store instance wired into deps
// (e.g. to assert on rollback behavior after a simulated failure).
func newProfileTestStore(t *testing.T) *profile.Store {
	t.Helper()
	preventRealHomeWrites(t)
	return profile.NewStore(filepath.Join(t.TempDir(), "profiles.json"))
}

// preventRealHomeWrites points XDG_CONFIG_HOME at a throwaway directory
// for the duration of the test. Code under test (resolveConfigDir in
// particular) can fall back to computing a real path under the user's home
// directory whenever a test forgets to pass --config-dir explicitly; this
// makes that a no-op instead of a test that writes outside its own sandbox
// onto whatever machine happens to run it.
func preventRealHomeWrites(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
}

// fakeDeps seeds a temp file backed profile.Store and an in-memory
// secret.Store, and returns a depsFunc wired to them, so command tests
// never touch the real Keychain or a real profiles.json.
func fakeDeps(t *testing.T, profiles []profile.Profile, secrets map[string]string) depsFunc {
	t.Helper()
	preventRealHomeWrites(t)

	store := profile.NewStore(filepath.Join(t.TempDir(), "profiles.json"))
	for _, p := range profiles {
		if p.CreatedAt.IsZero() {
			p.CreatedAt = time.Now()
		}
		if p.UpdatedAt.IsZero() {
			p.UpdatedAt = time.Now()
		}
		if err := store.Add(p); err != nil {
			t.Fatalf("seeding profile %q: %v", p.Name, err)
		}
	}

	mem := secret.NewMemStore()
	for name, value := range secrets {
		if err := mem.Set(name, value); err != nil {
			t.Fatalf("seeding secret for %q: %v", name, err)
		}
	}

	return func() (deps, error) {
		return deps{profiles: store, secrets: mem}, nil
	}
}
