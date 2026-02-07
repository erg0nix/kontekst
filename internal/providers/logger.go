package providers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/erg0nix/kontekst/internal/core"
)

type RequestLogger struct {
	logDir       string
	logRequests  bool
	logResponses bool
	logger       *slog.Logger
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

func NewRequestLogger(logDir string, logRequests, logResponses bool, logger *slog.Logger) *RequestLogger {
	return &RequestLogger{
		logDir:       logDir,
		logRequests:  logRequests,
		logResponses: logResponses,
		logger:       logger,
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
	l.logger.Debug("provider request", "request_id", requestID, "message_count", len(messages), "tool_count", len(tools))
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

	msgSummary := make([]string, 0, min(5, len(messages)))
	start := max(0, len(messages)-5)
	for i := start; i < len(messages); i++ {
		msg := messages[i]
		content := msg.Content
		if len(content) > 50 {
			content = content[:50] + "..."
		}
		msgSummary = append(msgSummary, fmt.Sprintf("[%s] %s", msg.Role, content))
	}

	l.logger.Error("provider request failed",
		"request_id", requestID,
		"status_code", statusCode,
		"error", string(errorBody),
		"recent_messages", msgSummary,
	)
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
