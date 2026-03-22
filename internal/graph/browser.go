package graph

import (
	"context"
	"os/exec"
	"runtime"
	"time"
)

// OpenBrowser attempts to open the given URL in the default browser.
// It fails silently on error (e.g., headless/SSH environment).
func OpenBrowser(url string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.CommandContext(ctx, "open", url)
	case "linux":
		cmd = exec.CommandContext(ctx, "xdg-open", url)
	case "windows":
		cmd = exec.CommandContext(ctx, "cmd", "/c", "start", url)
	default:
		return nil
	}
	return cmd.Start()
}
