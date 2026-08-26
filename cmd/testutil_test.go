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
	return profile.NewStore(filepath.Join(t.TempDir(), "profiles.json"))
}

// fakeDeps seeds a temp file backed profile.Store and an in-memory
// secret.Store, and returns a depsFunc wired to them, so command tests
// never touch the real Keychain or a real profiles.json.
func fakeDeps(t *testing.T, profiles []profile.Profile, secrets map[string]string) depsFunc {
	t.Helper()

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
