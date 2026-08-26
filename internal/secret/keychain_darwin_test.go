//go:build darwin

package secret

import (
	"os"
	"testing"
)

// These tests touch the real macOS Keychain, so they're opt in: run them
// explicitly with SANCTUM_TEST_KEYCHAIN=1. Plain `go test ./...` skips them
// so CI-like runs never depend on Keychain access or prompt for it.
func requireKeychainTests(t *testing.T) {
	t.Helper()
	if os.Getenv("SANCTUM_TEST_KEYCHAIN") == "" {
		t.Skip("skipping real Keychain test, set SANCTUM_TEST_KEYCHAIN=1 to run it")
	}
}

// testProfileName is namespaced so a test run can never collide with, or
// get mistaken for, a real sanctum profile's Keychain item.
const testProfileName = "sanctum-test-keychain-roundtrip"

func cleanupTestItem(t *testing.T, k *KeychainStore) {
	t.Helper()
	t.Cleanup(func() {
		_ = k.Delete(testProfileName)
	})
}

func TestKeychainSetGetDelete(t *testing.T) {
	requireKeychainTests(t)
	k := NewKeychainStore()
	cleanupTestItem(t, k)

	if err := k.Set(testProfileName, "sk-round-trip-value"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := k.Get(testProfileName)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "sk-round-trip-value" {
		t.Fatalf("got %q, want %q", got, "sk-round-trip-value")
	}

	if err := k.Delete(testProfileName); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if _, err := k.Get(testProfileName); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestKeychainSetOverwritesExisting(t *testing.T) {
	requireKeychainTests(t)
	k := NewKeychainStore()
	cleanupTestItem(t, k)

	if err := k.Set(testProfileName, "first-value"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := k.Set(testProfileName, "second-value"); err != nil {
		t.Fatalf("Set (overwrite): %v", err)
	}

	got, err := k.Get(testProfileName)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "second-value" {
		t.Fatalf("got %q, want %q", got, "second-value")
	}
}

func TestKeychainGetMissingReturnsNotFound(t *testing.T) {
	requireKeychainTests(t)
	k := NewKeychainStore()

	if _, err := k.Get("sanctum-test-keychain-definitely-missing"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestKeychainDeleteMissingIsNotAnError(t *testing.T) {
	requireKeychainTests(t)
	k := NewKeychainStore()

	if err := k.Delete("sanctum-test-keychain-definitely-missing"); err != nil {
		t.Fatalf("Delete of missing item should not error, got %v", err)
	}
}
