package store_test

import (
	"path/filepath"
	"testing"

	"sfpl-events-tui/store"
)

func TestLoad_nonExistentFile_returnsEmpty(t *testing.T) {
	dir := t.TempDir()
	store.OverridePath(filepath.Join(dir, "archived.json"))
	defer store.ResetPath()

	got, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty map, got %v", got)
	}
}

func TestSaveAndLoad_roundTrip(t *testing.T) {
	dir := t.TempDir()
	store.OverridePath(filepath.Join(dir, "archived.json"))
	defer store.ResetPath()

	input := map[string]bool{"Chess Club": true, "Book Club": true}
	if err := store.Save(input); err != nil {
		t.Fatal(err)
	}

	got, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !got["Chess Club"] || !got["Book Club"] {
		t.Errorf("round-trip failed: got %v", got)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 entries, got %d", len(got))
	}
}

func TestSave_createsDirectory(t *testing.T) {
	dir := t.TempDir()
	store.OverridePath(filepath.Join(dir, "nested", "dir", "archived.json"))
	defer store.ResetPath()

	if err := store.Save(map[string]bool{"X": true}); err != nil {
		t.Fatalf("Save should create missing dirs: %v", err)
	}
}
