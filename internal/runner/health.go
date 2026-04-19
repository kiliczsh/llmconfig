package runner

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// WaitHealthy polls GET /health until it returns 200 or the context is cancelled.
func WaitHealthy(ctx context.Context, host string, port int) error {
	url := fmt.Sprintf("http://%s:%d/health", host, port)
	client := &http.Client{Timeout: 3 * time.Second}

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	deadline := time.After(60 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline:
			return fmt.Errorf("health check timed out after 60s at %s", url)
		case <-ticker.C:
			resp, err := client.Get(url)
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					return nil
				}
			}
		}
	}
}
