package hosted

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// WaitForHostedHTTPReady polls until the generated MCP HTTP server answers GET (JSON info).
func WaitForHostedHTTPReady(ctx context.Context, dialHost, hostPort string) error {
	dialHost = strings.TrimSpace(dialHost)
	if dialHost == "" {
		return fmt.Errorf("dial host is empty")
	}
	hostPort = strings.TrimSpace(hostPort)
	if hostPort == "" {
		return fmt.Errorf("host port is empty")
	}
	baseURL := "http://" + net.JoinHostPort(dialHost, hostPort)
	deadline := time.Now().Add(hostedHTTPReadyTimeout)
	client := &http.Client{Timeout: 5 * time.Second}
	var lastErr error
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Accept", "application/json")
		resp, err := client.Do(req)
		if err == nil {
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 64*1024))
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
			lastErr = fmt.Errorf("HTTP %d from hosted runtime", resp.StatusCode)
		} else {
			lastErr = err
		}
		time.Sleep(hostedHTTPReadyPoll)
	}
	if lastErr != nil {
		return fmt.Errorf("hosted runtime did not become ready on %s within %v: %v", baseURL, hostedHTTPReadyTimeout, lastErr)
	}
	return fmt.Errorf("hosted runtime did not become ready on %s within %v", baseURL, hostedHTTPReadyTimeout)
}
