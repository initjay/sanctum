package profile

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	return NewStore(filepath.Join(dir, "profiles.json"))
}

func sampleProfile(name string) Profile {
	now := time.Now().UTC().Truncate(time.Second)
	return Profile{
		Name:           name,
		Label:          "test label",
		CredentialType: CredentialAPIKey,
		ConfigDir:      "/tmp/does-not-matter",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func TestLoadMissingFileReturnsEmpty(t *testing.T) {
	s := newTestStore(t)

	profiles, err := s.Load()
	if err != nil {
		t.Fatalf("Load on missing file returned error: %v", err)
	}
	if len(profiles) != 0 {
		t.Fatalf("expected 0 profiles, got %d", len(profiles))
	}
}

func TestAddAndGetRoundTrip(t *testing.T) {
	s := newTestStore(t)
	p := sampleProfile("work-acme")

	if err := s.Add(p); err != nil {
		t.Fatalf("Add: %v", err)
	}

	got, err := s.Get("work-acme")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != p.Name || got.Label != p.Label || got.CredentialType != p.CredentialType {
		t.Fatalf("round trip mismatch: got %+v, want %+v", got, p)
	}
}

func TestAddDuplicateNameFails(t *testing.T) {
	s := newTestStore(t)
	p := sampleProfile("dupe")

	if err := s.Add(p); err != nil {
		t.Fatalf("first Add: %v", err)
	}
	if err := s.Add(p); err != ErrAlreadyExists {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}
}

func TestGetMissingReturnsNotFound(t *testing.T) {
	s := newTestStore(t)

	if _, err := s.Get("nope"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestUpdateReplacesExisting(t *testing.T) {
	s := newTestStore(t)
	p := sampleProfile("editable")
	if err := s.Add(p); err != nil {
		t.Fatalf("Add: %v", err)
	}

	p.Label = "new label"
	if err := s.Update(p); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := s.Get("editable")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Label != "new label" {
		t.Fatalf("expected updated label, got %q", got.Label)
	}
}

func TestUpdateMissingReturnsNotFound(t *testing.T) {
	s := newTestStore(t)

	if err := s.Update(sampleProfile("ghost")); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestRemoveDeletesProfile(t *testing.T) {
	s := newTestStore(t)
	if err := s.Add(sampleProfile("removable")); err != nil {
		t.Fatalf("Add: %v", err)
	}

	if err := s.Remove("removable"); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	if _, err := s.Get("removable"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound after removal, got %v", err)
	}
}

func TestRemoveMissingReturnsNotFound(t *testing.T) {
	s := newTestStore(t)

	if err := s.Remove("ghost"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestSaveIsAtomicAndDoesNotLeakTempFiles(t *testing.T) {
	s := newTestStore(t)
	if err := s.Add(sampleProfile("a")); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := s.Add(sampleProfile("b")); err != nil {
		t.Fatalf("Add: %v", err)
	}

	entries, err := os.ReadDir(filepath.Dir(s.Path()))
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".tmp" {
			t.Fatalf("leftover temp file: %s", e.Name())
		}
	}

	info, err := os.Stat(s.Path())
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("expected profiles.json mode 0600, got %v", info.Mode().Perm())
	}
}

func TestValidName(t *testing.T) {
	valid := []string{"a", "work-acme", "work_acme", "Work123"}
	invalid := []string{"", "has space", "has/slash", "has.dot"}

	for _, name := range valid {
		if !ValidName(name) {
			t.Errorf("expected %q to be valid", name)
		}
	}
	for _, name := range invalid {
		if ValidName(name) {
			t.Errorf("expected %q to be invalid", name)
		}
	}
}
