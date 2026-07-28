package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAndSaveSecureStatePaths(t *testing.T) {
	base := t.TempDir()
	t.Setenv("XDG_STATE_HOME", base)
	dir := filepath.Join(base, "feeds-to-instapaper")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "added")
	if err := os.WriteFile(path, []byte("old\n"), 0644); err != nil {
		t.Fatal(err)
	}

	s, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if s.MarkProcessed("old") {
		t.Fatal("existing item was not loaded")
	}
	assertMode(t, dir, 0700)
	assertMode(t, path, 0600)

	s.Append("new")
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}
	assertMode(t, dir, 0700)
	assertMode(t, path, 0600)
}

func TestSaveCreatesSecureStatePaths(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "added")
	s := EmptyWithPath(path)
	s.Append("new")
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}
	assertMode(t, filepath.Dir(path), 0700)
	assertMode(t, path, 0600)
}

func assertMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("mode for %s = %o, want %o", path, got, want)
	}
}
