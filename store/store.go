package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
)

// defaultPath returns the canonical archive file path.
func defaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "archived.json"
	}
	return filepath.Join(home, ".local", "share", "sfpl-events-tui", "archived.json")
}

var overriddenPath string

// OverridePath replaces the storage path (for tests only).
func OverridePath(p string) { overriddenPath = p }

// ResetPath clears a test override.
func ResetPath() { overriddenPath = "" }

func path() string {
	if overriddenPath != "" {
		return overriddenPath
	}
	return defaultPath()
}

// Load reads the archived event names from disk. Returns an empty map if the
// file does not exist.
func Load() (map[string]bool, error) {
	data, err := os.ReadFile(path())
	if os.IsNotExist(err) {
		return map[string]bool{}, nil
	}
	if err != nil {
		return nil, err
	}
	var names []string
	if err := json.Unmarshal(data, &names); err != nil {
		return nil, err
	}
	m := make(map[string]bool, len(names))
	for _, n := range names {
		m[n] = true
	}
	return m, nil
}

// Save writes the archived event names to disk, creating directories as needed.
func Save(archived map[string]bool) error {
	p := path()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	names := make([]string, 0, len(archived))
	for n := range archived {
		names = append(names, n)
	}
	sort.Strings(names)
	data, err := json.Marshal(names)
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}
