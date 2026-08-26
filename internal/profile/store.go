package profile

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"syscall"
)

// ErrNotFound is returned when a profile name doesn't exist in the store.
var ErrNotFound = errors.New("profile not found")

// ErrAlreadyExists is returned when adding a profile whose name is already taken.
var ErrAlreadyExists = errors.New("profile already exists")

// ErrInvalidName is returned when a profile name fails ValidName.
var ErrInvalidName = errors.New("invalid profile name")

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
// same directory, fsyncs it, then renames it into place, so a crash mid-write
// can never leave profiles.json truncated or corrupt. It also fsyncs the
// containing directory afterward, since on most filesystems a rename isn't
// guaranteed durable until the directory entry itself is flushed.
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
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("syncing temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}

	if err := os.Rename(tmpPath, s.path); err != nil {
		return fmt.Errorf("replacing %s: %w", s.path, err)
	}

	if dirFile, err := os.Open(dir); err == nil {
		_ = dirFile.Sync()
		dirFile.Close()
	}

	return nil
}

// withLock runs fn with an exclusive lock held across the full
// load-mutate-save cycle, using a sidecar lock file. Without this, two
// concurrent sanctum invocations (same process or different processes) can
// each load the same snapshot, mutate it independently, and save, silently
// dropping one of the two changes.
func (s *Store) withLock(fn func(profiles []Profile) ([]Profile, error)) error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating %s: %w", dir, err)
	}

	lock, err := os.OpenFile(s.path+".lock", os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("opening lock file: %w", err)
	}
	defer lock.Close()

	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("locking %s: %w", lock.Name(), err)
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)

	profiles, err := s.Load()
	if err != nil {
		return err
	}

	updated, err := fn(profiles)
	if err != nil {
		return err
	}

	return s.save(updated)
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

// Add appends a new profile. It fails with ErrInvalidName if the name isn't
// acceptable, or ErrAlreadyExists if the name is already taken.
func (s *Store) Add(p Profile) error {
	if !ValidName(p.Name) {
		return fmt.Errorf("%w: %q", ErrInvalidName, p.Name)
	}

	return s.withLock(func(profiles []Profile) ([]Profile, error) {
		for _, existing := range profiles {
			if existing.Name == p.Name {
				return nil, ErrAlreadyExists
			}
		}
		return append(profiles, p), nil
	})
}

// Update replaces the profile with the given name. It fails with
// ErrInvalidName if the name isn't acceptable, or ErrNotFound if no such
// profile exists.
func (s *Store) Update(p Profile) error {
	if !ValidName(p.Name) {
		return fmt.Errorf("%w: %q", ErrInvalidName, p.Name)
	}

	return s.withLock(func(profiles []Profile) ([]Profile, error) {
		for i, existing := range profiles {
			if existing.Name == p.Name {
				profiles[i] = p
				return profiles, nil
			}
		}
		return nil, ErrNotFound
	})
}

// Remove deletes the profile with the given name. It fails with
// ErrNotFound if no such profile exists.
func (s *Store) Remove(name string) error {
	return s.withLock(func(profiles []Profile) ([]Profile, error) {
		for i, existing := range profiles {
			if existing.Name == name {
				return append(profiles[:i], profiles[i+1:]...), nil
			}
		}
		return nil, ErrNotFound
	})
}
