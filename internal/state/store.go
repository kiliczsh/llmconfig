package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kiliczsh/llamaconfig/internal/dirs"
)

type Store struct {
	statePath string
	lockPath  string
}

func NewStore() *Store {
	return &Store{
		statePath: dirs.StateFile(),
		lockPath:  dirs.StateLockFile(),
	}
}

// Load returns a snapshot of the state file without taking the lock. Safe
// for read-only callers because writes are atomic (temp file + rename).
// Callers that intend to mutate and write back MUST use Update instead —
// otherwise two concurrent read-modify-write cycles can lose each other's
// changes.
func (s *Store) Load() (*StateFile, error) {
	return s.loadUnlocked()
}

func (s *Store) loadUnlocked() (*StateFile, error) {
	raw, err := os.ReadFile(s.statePath)
	if os.IsNotExist(err) {
		return &StateFile{Version: 1, Models: map[string]*ModelState{}}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("state: read: %w", err)
	}

	var sf StateFile
	if err := json.Unmarshal(raw, &sf); err != nil {
		return nil, fmt.Errorf("state: parse: %w", err)
	}
	if sf.Models == nil {
		sf.Models = map[string]*ModelState{}
	}
	return &sf, nil
}

// Update atomically loads the state file, applies mutate, and writes the
// result back while holding the state lock across the entire operation.
// This prevents lost updates when multiple processes touch disjoint
// entries concurrently (e.g. `llamaconfig rm foo` racing with
// `llamaconfig up bar`).
//
// If mutate returns an error, the state file is left unchanged.
func (s *Store) Update(mutate func(*StateFile) error) error {
	lock, err := acquireLock(s.lockPath)
	if err != nil {
		return err
	}
	defer lock.release()

	sf, err := s.loadUnlocked()
	if err != nil {
		return err
	}
	if err := mutate(sf); err != nil {
		return err
	}
	return s.writeUnlocked(sf)
}

func (s *Store) writeUnlocked(sf *StateFile) error {
	data, err := json.MarshalIndent(sf, "", "  ")
	if err != nil {
		return fmt.Errorf("state: marshal: %w", err)
	}

	// Atomic write via temp file + rename.
	tmp := s.statePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("state: write tmp: %w", err)
	}
	if err := os.Rename(tmp, s.statePath); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("state: rename: %w", err)
	}
	return nil
}

// Get returns the state for a single model, or nil if no entry exists.
// Read-only — does not lock.
func (s *Store) Get(name string) (*ModelState, error) {
	sf, err := s.Load()
	if err != nil {
		return nil, err
	}
	ms, ok := sf.Models[name]
	if !ok {
		return nil, nil
	}
	return ms, nil
}

// Put atomically inserts or replaces a single model entry.
func (s *Store) Put(ms *ModelState) error {
	return s.Update(func(sf *StateFile) error {
		sf.Models[ms.Name] = ms
		return nil
	})
}

// Remove atomically deletes a single model entry.
func (s *Store) Remove(name string) error {
	return s.Update(func(sf *StateFile) error {
		delete(sf.Models, name)
		return nil
	})
}

// EnsureDir makes sure the state file directory exists.
func (s *Store) EnsureDir() error {
	return os.MkdirAll(filepath.Dir(s.statePath), 0755)
}
