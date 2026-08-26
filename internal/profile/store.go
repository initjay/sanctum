package profile

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

// ErrNotFound is returned when a profile name doesn't exist in the store.
var ErrNotFound = errors.New("profile not found")

// ErrAlreadyExists is returned when adding a profile whose name is already taken.
var ErrAlreadyExists = errors.New("profile already exists")

var nameRE = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// ValidName reports whether name is an acceptable profile name.
func ValidName(name string) bool {
	return name != "" && nameRE.MatchString(name)
}

// Store persists profile metadata to a JSON file on disk. It never holds a
// raw secret value, only Profile structs.
type Store struct {
	path string
}

// NewStore returns a Store backed by the file at path. The file and its
// parent directory are created on first Save if they don't already exist.
func NewStore(path string) *Store {
	return &Store{path: path}
}

// Path returns the underlying file path this store reads and writes.
func (s *Store) Path() string {
	return s.path
}

// Load reads all profiles from disk. A missing file is treated as an empty
// store rather than an error, so callers don't need a separate bootstrap step.
func (s *Store) Load() ([]Profile, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return []Profile{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", s.path, err)
	}

	var pf profilesFile
	if err := json.Unmarshal(data, &pf); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", s.path, err)
	}

	if pf.Profiles == nil {
		return []Profile{}, nil
	}

	return pf.Profiles, nil
}

// save writes profiles to disk atomically: it writes to a temp file in the
// same directory, then renames it into place, so a crash mid-write can never
// leave profiles.json truncated or corrupt.
func (s *Store) save(profiles []Profile) error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating %s: %w", dir, err)
	}

	pf := profilesFile{Version: currentProfilesVersion, Profiles: profiles}
	data, err := json.MarshalIndent(pf, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding profiles: %w", err)
	}
	data = append(data, '\n')

	tmp, err := os.CreateTemp(dir, ".profiles-*.json.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return fmt.Errorf("setting permissions on temp file: %w", err)
	}

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("writing temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}

	if err := os.Rename(tmpPath, s.path); err != nil {
		return fmt.Errorf("replacing %s: %w", s.path, err)
	}

	return nil
}

// Get returns the profile with the given name, or ErrNotFound.
func (s *Store) Get(name string) (Profile, error) {
	profiles, err := s.Load()
	if err != nil {
		return Profile{}, err
	}

	for _, p := range profiles {
		if p.Name == name {
			return p, nil
		}
	}

	return Profile{}, ErrNotFound
}

// Add appends a new profile. It fails with ErrAlreadyExists if the name is
// already taken.
func (s *Store) Add(p Profile) error {
	profiles, err := s.Load()
	if err != nil {
		return err
	}

	for _, existing := range profiles {
		if existing.Name == p.Name {
			return ErrAlreadyExists
		}
	}

	profiles = append(profiles, p)
	return s.save(profiles)
}

// Update replaces the profile with the given name. It fails with
// ErrNotFound if no such profile exists.
func (s *Store) Update(p Profile) error {
	profiles, err := s.Load()
	if err != nil {
		return err
	}

	for i, existing := range profiles {
		if existing.Name == p.Name {
			profiles[i] = p
			return s.save(profiles)
		}
	}

	return ErrNotFound
}

// Remove deletes the profile with the given name. It fails with
// ErrNotFound if no such profile exists.
func (s *Store) Remove(name string) error {
	profiles, err := s.Load()
	if err != nil {
		return err
	}

	for i, existing := range profiles {
		if existing.Name == name {
			profiles = append(profiles[:i], profiles[i+1:]...)
			return s.save(profiles)
		}
	}

	return ErrNotFound
}
