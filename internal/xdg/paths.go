// Package xdg resolves where sanctum keeps its own state on disk, separate
// from anything Claude Code itself manages.
package xdg

import (
	"os"
	"path/filepath"
)

// ConfigHome returns the base config directory sanctum should use, honoring
// XDG_CONFIG_HOME if set and falling back to ~/.config otherwise.
func ConfigHome() (string, error) {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return dir, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".config"), nil
}

// SanctumHome returns ~/.config/sanctum (or the XDG_CONFIG_HOME equivalent),
// creating it if it doesn't already exist.
func SanctumHome() (string, error) {
	base, err := ConfigHome()
	if err != nil {
		return "", err
	}

	dir := filepath.Join(base, "sanctum")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}

	return dir, nil
}

// ProfilesFile returns the path to sanctum's profile metadata file.
func ProfilesFile() (string, error) {
	home, err := SanctumHome()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, "profiles.json"), nil
}

// ClaudeHomesDir returns the directory sanctum uses to hold each profile's
// isolated CLAUDE_CONFIG_DIR.
func ClaudeHomesDir() (string, error) {
	home, err := SanctumHome()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, "claude-homes"), nil
}
