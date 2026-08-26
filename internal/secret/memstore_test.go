package secret

import "testing"

func TestMemStoreSetAndGet(t *testing.T) {
	m := NewMemStore()

	if err := m.Set("work-acme", "sk-test-123"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := m.Get("work-acme")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "sk-test-123" {
		t.Fatalf("got %q, want %q", got, "sk-test-123")
	}
}

func TestMemStoreGetMissingReturnsNotFound(t *testing.T) {
	m := NewMemStore()

	if _, err := m.Get("nope"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestMemStoreSetOverwrites(t *testing.T) {
	m := NewMemStore()

	if err := m.Set("work-acme", "first"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := m.Set("work-acme", "second"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := m.Get("work-acme")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "second" {
		t.Fatalf("got %q, want %q", got, "second")
	}
}

func TestMemStoreDeleteRemovesValue(t *testing.T) {
	m := NewMemStore()
	if err := m.Set("work-acme", "value"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	if err := m.Delete("work-acme"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if _, err := m.Get("work-acme"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestMemStoreDeleteMissingIsNotAnError(t *testing.T) {
	m := NewMemStore()

	if err := m.Delete("never-existed"); err != nil {
		t.Fatalf("Delete of missing key should not error, got %v", err)
	}
}
