package state

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
)

type State struct {
	Path           string
	ProcessedItems sync.Map
	NewItems       []string
}

func EmptyWithPath(path string) *State {
	return &State{
		Path:           path,
		ProcessedItems: sync.Map{},
		NewItems:       make([]string, 0),
	}
}

func New() *State {
	stateDir := os.Getenv("XDG_STATE_HOME")
	if stateDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("Failed to get user home directory: %v", err)
		}
		stateDir = filepath.Join(homeDir, ".local", "state")
	}

	return EmptyWithPath(filepath.Join(stateDir, "feeds-to-instapaper", "added"))
}

func Load() (*State, error) {
	state := New()

	if err := ensureDirectory(filepath.Dir(state.Path)); err != nil {
		return nil, err
	}

	info, err := os.Stat(state.Path)
	if os.IsNotExist(err) {
		return state, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to stat state file %s: %w", state.Path, err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("state file %s must be a regular file", state.Path)
	}
	if err := os.Chmod(state.Path, 0600); err != nil {
		return nil, fmt.Errorf("failed to secure state file %s: %w", state.Path, err)
	}

	file, err := os.Open(state.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open state file %s: %w", state.Path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		state.MarkProcessed(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read state file %s: %w", state.Path, err)
	}

	return state, nil
}

func (s *State) Save() error {
	if len(s.NewItems) < 1 {
		return nil
	}

	if err := ensureDirectory(filepath.Dir(s.Path)); err != nil {
		return err
	}

	file, err := os.OpenFile(s.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to create state file: %w", err)
	}
	defer file.Close()
	if err := file.Chmod(0600); err != nil {
		return fmt.Errorf("failed to secure state file: %w", err)
	}

	for _, item := range s.NewItems {
		if _, err := file.WriteString(item + "\n"); err != nil {
			return fmt.Errorf("failed to write state file: %w", err)
		}
	}

	return nil
}

func ensureDirectory(path string) error {
	if err := os.MkdirAll(path, 0700); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}
	if err := os.Chmod(path, 0700); err != nil {
		return fmt.Errorf("failed to secure state directory: %w", err)
	}
	return nil
}

func (s *State) Append(item string) {
	s.NewItems = append(s.NewItems, item)
}

// MarkProcessed returns true if the item has not been marked before; otherwise, returns false.
func (s *State) MarkProcessed(item string) bool {
	_, loaded := s.ProcessedItems.LoadOrStore(item, struct{}{})
	return !loaded
}
