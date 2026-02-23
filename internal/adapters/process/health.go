package process

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// WaitForHealth polls the /healthz endpoint until it returns 200 or the timeout expires.
func WaitForHealth(ctx context.Context, baseURL string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	interval := 200 * time.Millisecond
	client := &http.Client{Timeout: 2 * time.Second}

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/healthz", nil)
		if err != nil {
			return fmt.Errorf("create health request: %w", err)
		}

		resp, err := client.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}

		time.Sleep(interval)
		// Exponential backoff up to 2 seconds
		if interval < 2*time.Second {
			interval = interval * 3 / 2
		}
	}

	return fmt.Errorf("health check timed out after %s", timeout)
}
