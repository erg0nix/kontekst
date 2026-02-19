package builtin

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/erg0nix/kontekst/internal/config"
	toolpkg "github.com/erg0nix/kontekst/internal/tool"
)

// WebFetch is a tool that fetches content from URLs via HTTP.
type WebFetch struct {
	WebConfig config.WebToolsConfig
}

func (tool *WebFetch) Name() string { return "web_fetch" }
func (tool *WebFetch) Description() string {
	return "Fetches content from a URL using HTTP GET. Returns the response body as text."
}
func (tool *WebFetch) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url": map[string]any{
				"type":        "string",
				"description": "The URL to fetch",
			},
			"method": map[string]any{
				"type":        "string",
				"description": "HTTP method (default: GET)",
			},
			"headers": map[string]any{
				"type":        "object",
				"description": "Additional headers to include",
			},
		},
		"required": []string{"url"},
	}
}
func (tool *WebFetch) RequiresApproval() bool { return true }

func (tool *WebFetch) Execute(args map[string]any, ctx context.Context) (string, error) {
	url, ok := getStringArg("url", args)
	if !ok || url == "" {
		return "", errors.New("missing url")
	}

	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return "", errors.New("url must start with http:// or https://")
	}

	method, _ := getStringArg("method", args)
	if method == "" {
		method = "GET"
	}
	method = strings.ToUpper(method)

	if method != "GET" && method != "HEAD" {
		return "", errors.New("only GET and HEAD methods are supported")
	}

	timeout := time.Duration(tool.WebConfig.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "kontekst/1.0")

	if headers, ok := args["headers"].(map[string]any); ok {
		for key, value := range headers {
			if strValue, ok := value.(string); ok {
				req.Header.Set(key, strValue)
			}
		}
	}

	client := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return errors.New("too many redirects")
			}
			return nil
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	maxBytes := tool.WebConfig.MaxResponseBytes
	if maxBytes <= 0 {
		maxBytes = 5 * 1024 * 1024
	}

	limitedReader := io.LimitReader(resp.Body, maxBytes+1)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	truncated := false
	if int64(len(body)) > maxBytes {
		body = body[:maxBytes]
		truncated = true
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Status: %s\n", resp.Status))
	result.WriteString(fmt.Sprintf("Content-Type: %s\n", resp.Header.Get("Content-Type")))
	result.WriteString(fmt.Sprintf("Content-Length: %d\n", len(body)))
	if truncated {
		result.WriteString("(Response truncated due to size limit)\n")
	}
	result.WriteString("\n")
	result.Write(body)

	return result.String(), nil
}

// RegisterWebFetch adds the web_fetch tool to the registry.
func RegisterWebFetch(registry *toolpkg.Registry, webConfig config.WebToolsConfig) {
	registry.Add(&WebFetch{WebConfig: webConfig})
}
