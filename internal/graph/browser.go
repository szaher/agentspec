package graph

import (
	"os/exec"
	"runtime"
)

// OpenBrowser attempts to open the given URL in the default browser.
// It fails silently on error (e.g., headless/SSH environment).
func OpenBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return nil
	}
	return cmd.Start()
}
