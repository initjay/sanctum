package cmd

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/initjay/sanctum/internal/profile"
	"github.com/initjay/sanctum/internal/secret"
)

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
