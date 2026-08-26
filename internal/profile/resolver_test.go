package profile

import (
	"os"
	"testing"
	"time"

	"github.com/initjay/sanctum/internal/secret"
)

func newResolverFixtures(t *testing.T) (*Store, *secret.MemStore) {
	t.Helper()
	return newTestStore(t), secret.NewMemStore()
}

func TestResolveProfileAPIKey(t *testing.T) {
	store, secrets := newResolverFixtures(t)

	p := Profile{
		Name:           "work-acme",
		Label:          "ACME",
		CredentialType: CredentialAPIKey,
		ConfigDir:      "/home/work-acme",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	if err := store.Add(p); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := secrets.Set("work-acme", "sk-secret"); err != nil {
		t.Fatalf("Set secret: %v", err)
	}

	resolved, err := ResolveProfile(store, secrets, "work-acme")
	if err != nil {
		t.Fatalf("ResolveProfile: %v", err)
	}

	if resolved.ProfileName != "work-acme" {
		t.Errorf("expected profile name work-acme, got %q", resolved.ProfileName)
	}
	if resolved.ConfigDir != "/home/work-acme" {
		t.Errorf("expected config dir /home/work-acme, got %q", resolved.ConfigDir)
	}
	if resolved.Vars[EnvAnthropicAPIKey] != "sk-secret" {
		t.Errorf("expected resolved api key, got %q", resolved.Vars[EnvAnthropicAPIKey])
	}
	if !contains(resolved.UnsetVars, EnvClaudeOAuthToken) {
		t.Errorf("expected oauth token var in unset list")
	}
}

func TestResolveProfileRejectsInvalidCredentialTypeOnDisk(t *testing.T) {
	// Store.Add/Update can't produce this, so simulate a hand edited or
	// corrupted profiles.json by writing the file directly, bypassing the
	// store's own validation entirely.
	store, secrets := newResolverFixtures(t)
	raw := `{"version":1,"profiles":[{"name":"corrupted","credential_type":"not-a-real-type","config_dir":"/home/corrupted"}]}`
	if err := os.WriteFile(store.Path(), []byte(raw), 0o600); err != nil {
		t.Fatalf("writing corrupted profiles.json: %v", err)
	}
	if err := secrets.Set("corrupted", "sk-should-not-be-used"); err != nil {
		t.Fatalf("Set secret: %v", err)
	}

	if _, err := ResolveProfile(store, secrets, "corrupted"); err == nil {
		t.Fatalf("expected an error resolving a profile with an invalid credential type, got nil")
	}
}

func TestResolveProfileUnknownName(t *testing.T) {
	store, secrets := newResolverFixtures(t)

	if _, err := ResolveProfile(store, secrets, "ghost"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestResolveProfileMissingSecret(t *testing.T) {
	store, secrets := newResolverFixtures(t)

	p := Profile{
		Name:           "no-secret",
		CredentialType: CredentialAPIKey,
		ConfigDir:      "/home/no-secret",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	if err := store.Add(p); err != nil {
		t.Fatalf("Add: %v", err)
	}

	if _, err := ResolveProfile(store, secrets, "no-secret"); err != secret.ErrNotFound {
		t.Fatalf("expected secret.ErrNotFound, got %v", err)
	}
}
