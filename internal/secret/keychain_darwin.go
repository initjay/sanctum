//go:build darwin

package secret

import (
	"fmt"

	keychain "github.com/keybase/go-keychain"
)

// keychainService namespaces every sanctum Keychain item under its own
// service name, so they never collide with Claude Code's own Keychain
// items or anything else on the machine.
const keychainService = "sanctum"

// KeychainStore stores secrets in the macOS Keychain.
type KeychainStore struct{}

var _ Store = (*KeychainStore)(nil)

// NewKeychainStore returns a Store backed by the macOS Keychain.
func NewKeychainStore() *KeychainStore {
	return &KeychainStore{}
}

func (k *KeychainStore) Get(profileName string) (string, error) {
	data, err := keychain.GetGenericPassword(keychainService, profileName, "", "")
	if err != nil {
		return "", fmt.Errorf("reading keychain item for %q: %w", profileName, err)
	}
	if data == nil {
		return "", ErrNotFound
	}

	return string(data), nil
}

func (k *KeychainStore) Set(profileName, value string) error {
	// A profile has exactly one secret at a time, so overwriting means
	// delete-then-add rather than a partial attribute update. The delete is
	// expected to fail with "item not found" the first time a profile's
	// secret is set, which is fine, there's nothing to remove yet.
	if err := keychain.DeleteGenericPasswordItem(keychainService, profileName); err != nil && err != keychain.ErrorItemNotFound {
		return fmt.Errorf("clearing existing keychain item for %q: %w", profileName, err)
	}

	item := keychain.NewGenericPassword(keychainService, profileName, "sanctum: "+profileName, []byte(value), "")
	item.SetAccessible(keychain.AccessibleWhenUnlockedThisDeviceOnly)
	item.SetSynchronizable(keychain.SynchronizableNo)

	if err := keychain.AddItem(item); err != nil {
		return fmt.Errorf("writing keychain item for %q: %w", profileName, err)
	}

	return nil
}

func (k *KeychainStore) Delete(profileName string) error {
	if err := keychain.DeleteGenericPasswordItem(keychainService, profileName); err != nil && err != keychain.ErrorItemNotFound {
		return fmt.Errorf("deleting keychain item for %q: %w", profileName, err)
	}

	return nil
}
