package runner

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// healthURL returns the appropriate health-check endpoint for each backend.
func healthURL(backend, host string, port int) string {
	base := fmt.Sprintf("http://%s:%d", host, port)
	switch backend {
	case "sd":
		return base + "/"
	case "whisper":
		return base + "/"
	default:
		return base + "/health"
	}
}

// healthTimeout returns how long to wait for each backend to become ready.
func healthTimeout(backend string) time.Duration {
	switch backend {
	case "sd":
		return 120 * time.Second // SD models are slow to load into VRAM
	case "whisper":
		return 30 * time.Second
	default:
		return 60 * time.Second
	}
}

// WaitHealthy polls the backend's health endpoint until it returns 200 or times out.
func WaitHealthy(ctx context.Context, host string, port int, backend string) error {
	url := healthURL(backend, host, port)
	timeout := healthTimeout(backend)
	client := &http.Client{Timeout: 3 * time.Second}

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	deadline := time.After(timeout)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline:
			return fmt.Errorf("health check timed out after %.0fs at %s", timeout.Seconds(), url)
		case <-ticker.C:
			resp, err := client.Get(url)
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode < 500 {
					return nil
				}
			}
		}
	}
}
