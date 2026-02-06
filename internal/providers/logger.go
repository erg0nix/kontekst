package providers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/erg0nix/kontekst/internal/core"
)

type RequestLogger struct {
	logDir       string
	logRequests  bool
	logResponses bool
}

type LogEntry struct {
	Timestamp  string               `json:"timestamp"`
	RequestID  string               `json:"request_id"`
	Type       string               `json:"type"`
	Messages   []core.Message       `json:"messages,omitempty"`
	Tools      []core.ToolDef       `json:"tools,omitempty"`
	Sampling   *core.SamplingConfig `json:"sampling,omitempty"`
	Payload    map[string]any       `json:"payload,omitempty"`
	Response   *core.ChatResponse   `json:"response,omitempty"`
	Duration   string               `json:"duration,omitempty"`
	Error      string               `json:"error,omitempty"`
	StatusCode int                  `json:"status_code,omitempty"`
}

func NewRequestLogger(logDir string, logRequests, logResponses bool) *RequestLogger {
	return &RequestLogger{
		logDir:       logDir,
		logRequests:  logRequests,
		logResponses: logResponses,
	}
}

func (l *RequestLogger) LogRequest(requestID core.RequestID, messages []core.Message, tools []core.ToolDef, sampling *core.SamplingConfig, payload map[string]any) {
	if !l.logRequests {
		return
	}

	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		RequestID: string(requestID),
		Type:      "request",
		Messages:  messages,
		Tools:     tools,
		Sampling:  sampling,
		Payload:   payload,
	}

	l.writeLog(entry)
	l.printToConsole("REQUEST", requestID, entry)
}

func (l *RequestLogger) LogResponse(requestID core.RequestID, response core.ChatResponse, duration time.Duration) {
	if !l.logResponses {
		return
	}

	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		RequestID: string(requestID),
		Type:      "response",
		Response:  &response,
		Duration:  duration.String(),
	}

	l.writeLog(entry)
}

func (l *RequestLogger) LogError(requestID core.RequestID, statusCode int, errorBody []byte, messages []core.Message, payload map[string]any) {
	entry := LogEntry{
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		RequestID:  string(requestID),
		Type:       "error",
		StatusCode: statusCode,
		Error:      string(errorBody),
		Messages:   messages,
		Payload:    payload,
	}

	l.writeLog(entry)
	l.printErrorToConsole(requestID, statusCode, errorBody, messages)
}

func (l *RequestLogger) writeLog(entry LogEntry) {
	if l.logDir == "" {
		return
	}

	_ = os.MkdirAll(l.logDir, 0o755)

	logFile := filepath.Join(l.logDir, fmt.Sprintf("provider_%s.jsonl", time.Now().Format("2006-01-02")))

	data, _ := json.Marshal(entry)
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()

	_, _ = f.Write(data)
	_, _ = f.WriteString("\n")
}

func (l *RequestLogger) printToConsole(prefix string, requestID core.RequestID, entry LogEntry) {
	if l.logRequests {
		data, _ := json.MarshalIndent(entry, "", "  ")
		fmt.Fprintf(os.Stderr, "\n[%s %s]\n%s\n\n", prefix, requestID, string(data))
	}
}

func (l *RequestLogger) printErrorToConsole(requestID core.RequestID, statusCode int, errorBody []byte, messages []core.Message) {
	fmt.Fprintf(os.Stderr, "\n[ERROR] Provider request failed (request_id=%s)\n", requestID)
	fmt.Fprintf(os.Stderr, "Status: %d\n", statusCode)
	fmt.Fprintf(os.Stderr, "Response: %s\n\n", string(errorBody))

	fmt.Fprintf(os.Stderr, "Message sequence (last %d):\n", min(5, len(messages)))
	start := max(0, len(messages)-5)
	for i := start; i < len(messages); i++ {
		msg := messages[i]
		fmt.Fprintf(os.Stderr, "  %d. [%s] ", i, msg.Role)
		if len(msg.Content) > 50 {
			fmt.Fprintf(os.Stderr, "%s...", msg.Content[:50])
		} else if msg.Content != "" {
			fmt.Fprintf(os.Stderr, "%s", msg.Content)
		}
		if len(msg.ToolCalls) > 0 {
			fmt.Fprintf(os.Stderr, " (%d tool calls)", len(msg.ToolCalls))
		}
		if msg.ToolResult != nil {
			fmt.Fprintf(os.Stderr, " tool result: %s", msg.ToolResult.Name)
		}
		fmt.Fprintf(os.Stderr, "\n")
	}
	fmt.Fprintf(os.Stderr, "\n")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
