// Package profile holds sanctum's profile metadata model and the storage
// layer for it. It intentionally has no field anywhere for a raw secret
// value: secrets live only in the secret.Store implementation, referenced
// by profile name.
package profile

import "time"

// CredentialType identifies which Claude Code env var a profile's secret
// should be injected as.
type CredentialType string

const (
	// CredentialAPIKey maps to ANTHROPIC_API_KEY, for Console/org API tokens.
	CredentialAPIKey CredentialType = "api_key"
	// CredentialOAuthToken maps to CLAUDE_CODE_OAUTH_TOKEN, minted via
	// `claude setup-token` for subscription accounts.
	CredentialOAuthToken CredentialType = "oauth_token"
)

// Valid reports whether c is one of the known credential types.
func (c CredentialType) Valid() bool {
	switch c {
	case CredentialAPIKey, CredentialOAuthToken:
		return true
	default:
		return false
	}
}

// Profile is the metadata sanctum keeps for one isolated Claude identity.
// The actual secret is never stored here, only referenced by Name.
type Profile struct {
	Name           string         `json:"name"`
	Label          string         `json:"label"`
	CredentialType CredentialType `json:"credential_type"`
	ConfigDir      string         `json:"config_dir"`
	BaseURL        string         `json:"base_url,omitempty"`
	DefaultModel   string         `json:"default_model,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

// profilesFile is the on-disk shape of profiles.json.
type profilesFile struct {
	Version  int       `json:"version"`
	Profiles []Profile `json:"profiles"`
}

const currentProfilesVersion = 1
