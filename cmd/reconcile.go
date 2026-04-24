package cmd

import (
	"github.com/kiliczsh/llmconfig/internal/output"
	"github.com/kiliczsh/llmconfig/internal/runner"
	"github.com/kiliczsh/llmconfig/internal/state"
)

// reconcileLiveness loads the state file, marks any "running" entry whose
// backing process is no longer alive as "stopped", and persists the result
// in a single atomic update. It returns the post-reconcile snapshot.
//
// Read-only commands (ps, models) call this so users see accurate status
// without having to run `down` or `status` to flush stale entries.
func reconcileLiveness(store *state.Store, r runner.Runner, p *output.Printer) (*state.StateFile, error) {
	var result *state.StateFile
	err := store.Update(func(sf *state.StateFile) error {
		for _, ms := range sf.Models {
			if ms.Status == "running" && !r.IsAlive(ms) {
				ms.Status = "stopped"
			}
		}
		result = sf
		return nil
	})
	if err == nil {
		return result, nil
	}

	// Write failures shouldn't block the display; fall back to an in-memory
	// reconciliation on a fresh read-only snapshot.
	if p != nil {
		p.Warn("could not persist reconciled state: %v", err)
	}
	sf, lerr := store.Load()
	if lerr != nil {
		return nil, lerr
	}
	for _, ms := range sf.Models {
		if ms.Status == "running" && !r.IsAlive(ms) {
			ms.Status = "stopped"
		}
	}
	return sf, nil
}
