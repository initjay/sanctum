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
	// Try an in place update first, since that's a single atomic
	// SecItemUpdate call: at every point in time the item either still holds
	// the old value or already holds the new one, never neither. A
	// delete-then-add would leave a real gap where the profile has no
	// stored secret at all if the process died or another Set raced in
	// between the two calls, and that gap is exactly what this tool exists
	// to avoid.
	query := keychain.NewItem()
	query.SetSecClass(keychain.SecClassGenericPassword)
	query.SetService(keychainService)
	query.SetAccount(profileName)

	update := keychain.NewItem()
	update.SetData([]byte(value))
	update.SetLabel("sanctum: " + profileName)
	update.SetAccessible(keychain.AccessibleWhenUnlockedThisDeviceOnly)
	update.SetSynchronizable(keychain.SynchronizableNo)

	err := keychain.UpdateItem(query, update)
	if err == nil {
		return nil
	}
	if err != keychain.ErrorItemNotFound {
		return fmt.Errorf("updating keychain item for %q: %w", profileName, err)
	}

	// Nothing to update yet, this is the first time this profile's secret
	// is being set.
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
