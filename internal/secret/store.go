// Package secret defines where sanctum keeps the actual credential values
// (API keys, OAuth tokens) for each profile, separate from profile metadata.
package secret

import "errors"

// ErrNotFound is returned when no secret is stored for a given profile name.
var ErrNotFound = errors.New("secret not found")

// Store holds one secret value per profile name.
type Store interface {
	// Get returns the secret for profileName, or ErrNotFound.
	Get(profileName string) (string, error)
	// Set stores value as the secret for profileName, overwriting any
	// existing value.
	Set(profileName, value string) error
	// Delete removes the secret for profileName. It is not an error to
	// delete a profile that has no stored secret.
	Delete(profileName string) error
}
