package secret

import "sync"

// MemStore is an in-memory Store, used by tests and anywhere a real
// Keychain isn't available or wanted.
type MemStore struct {
	mu     sync.Mutex
	values map[string]string
}

// NewMemStore returns an empty MemStore.
func NewMemStore() *MemStore {
	return &MemStore{values: make(map[string]string)}
}

func (m *MemStore) Get(profileName string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	value, ok := m.values[profileName]
	if !ok {
		return "", ErrNotFound
	}
	return value, nil
}

func (m *MemStore) Set(profileName, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.values[profileName] = value
	return nil
}

func (m *MemStore) Delete(profileName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.values, profileName)
	return nil
}
