package runner

import (
	"context"
	"time"

	"github.com/kiliczsh/llamaconfig/internal/config"
	"github.com/kiliczsh/llamaconfig/internal/state"
)

// Runner manages the llama-server process lifecycle.
type Runner interface {
	Start(ctx context.Context, rc *config.RunConfig) (*state.ModelState, error)
	Stop(ctx context.Context, ms *state.ModelState, timeout time.Duration) error
	IsAlive(ms *state.ModelState) bool
}

// New returns the default Runner implementation.
func New() Runner {
	return &serverRunner{}
}
