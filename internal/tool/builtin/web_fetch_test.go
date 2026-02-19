package builtin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/erg0nix/kontekst/internal/config"
)

func TestWebFetchExecute(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte("hello world"))
		case "/json":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"status": "ok"}`))
		case "/large":
			w.Header().Set("Content-Type", "text/plain")
			for i := 0; i < 1000; i++ {
				w.Write([]byte("large response content\n"))
			}
		case "/error":
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("server error"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	tool := &WebFetch{WebConfig: config.WebToolsConfig{
		TimeoutSeconds:   30,
		MaxResponseBytes: 1000,
	}}

	tests := []struct {
		name    string
		args    map[string]any
		wantHas []string
		wantErr bool
	}{
		{
			name:    "successful GET",
			args:    map[string]any{"url": server.URL + "/ok"},
			wantHas: []string{"Status: 200", "hello world"},
		},
		{
			name:    "JSON response",
			args:    map[string]any{"url": server.URL + "/json"},
			wantHas: []string{"application/json", "status"},
		},
		{
			name:    "response truncated",
			args:    map[string]any{"url": server.URL + "/large"},
			wantHas: []string{"truncated"},
		},
		{
			name:    "server error returns body",
			args:    map[string]any{"url": server.URL + "/error"},
			wantHas: []string{"500"},
		},
		{
			name:    "missing url",
			args:    map[string]any{},
			wantErr: true,
		},
		{
			name:    "invalid url scheme",
			args:    map[string]any{"url": "ftp://example.com"},
			wantErr: true,
		},
		{
			name:    "unsupported method",
			args:    map[string]any{"url": server.URL + "/ok", "method": "POST"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tool.Execute(tt.args, context.Background())

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got result: %s", result)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			for _, want := range tt.wantHas {
				if !strings.Contains(result, want) {
					t.Errorf("result should contain %q, got:\n%s", want, result)
				}
			}
		})
	}
}

func TestWebFetchHead(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "HEAD" {
			t.Errorf("expected HEAD method, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Content-Length", "100")
	}))
	defer server.Close()

	tool := &WebFetch{}
	result, err := tool.Execute(map[string]any{
		"url":    server.URL,
		"method": "HEAD",
	}, context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "Status: 200") {
		t.Errorf("HEAD request failed, got: %s", result)
	}
}

func TestWebFetchHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Custom") != "test-value" {
			t.Errorf("expected custom header, got %q", r.Header.Get("X-Custom"))
		}
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	tool := &WebFetch{}
	_, err := tool.Execute(map[string]any{
		"url":     server.URL,
		"headers": map[string]any{"X-Custom": "test-value"},
	}, context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWebFetchUserAgent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("User-Agent"), "kontekst") {
			t.Errorf("expected kontekst user agent, got %q", r.Header.Get("User-Agent"))
		}
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	tool := &WebFetch{}
	_, err := tool.Execute(map[string]any{"url": server.URL}, context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
