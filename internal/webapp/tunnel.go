package webapp

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"regexp"
	"time"
)

var tunnelURLPattern = regexp.MustCompile(`https://[a-z0-9-]+\.trycloudflare\.com`)

// StartTunnel launches a cloudflared quick tunnel pointing to the local server.
// Returns the public HTTPS URL. The tunnel process lives until ctx is cancelled.
func StartTunnel(ctx context.Context, port int, logger *slog.Logger) (string, error) {
	cmd := exec.CommandContext(ctx, "cloudflared", "tunnel", "--url", fmt.Sprintf("http://localhost:%d", port))

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("webapp: tunnel stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("webapp: start cloudflared: %w", err)
	}

	// Parse URL from stderr output.
	urlCh := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			logger.Debug("cloudflared", "line", line)

			if match := tunnelURLPattern.FindString(line); match != "" {
				urlCh <- match
				return
			}
		}
	}()

	// Wait for URL with timeout.
	select {
	case url := <-urlCh:
		logger.Info("tunnel started", "url", url)
		return url, nil
	case <-time.After(30 * time.Second):
		cmd.Process.Kill()
		return "", fmt.Errorf("webapp: tunnel URL not found within 30s")
	case <-ctx.Done():
		return "", ctx.Err()
	}
}
