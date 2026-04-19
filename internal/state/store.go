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

func (s *Store) Load() (*StateFile, error) {
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

func (s *Store) Save(sf *StateFile) error {
	lock, err := acquireLock(s.lockPath)
	if err != nil {
		return err
	}
	defer lock.release()

	data, err := json.MarshalIndent(sf, "", "  ")
	if err != nil {
		return fmt.Errorf("state: marshal: %w", err)
	}

	// Atomic write via temp file + rename
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

func (s *Store) Put(ms *ModelState) error {
	sf, err := s.Load()
	if err != nil {
		return err
	}
	sf.Models[ms.Name] = ms
	return s.Save(sf)
}

func (s *Store) Remove(name string) error {
	sf, err := s.Load()
	if err != nil {
		return err
	}
	delete(sf.Models, name)
	return s.Save(sf)
}

// EnsureDir makes sure the state file directory exists.
func (s *Store) EnsureDir() error {
	return os.MkdirAll(filepath.Dir(s.statePath), 0755)
}
