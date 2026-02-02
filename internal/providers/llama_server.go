package providers

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/erg0nix/kontekst/internal/config"
	"github.com/erg0nix/kontekst/internal/core"
)

type LlamaServerProvider struct {
	cfg    config.LlamaServerConfig
	client *http.Client
	mu     sync.Mutex
	cmd    *exec.Cmd
	start  time.Time
}

func NewLlamaServerProvider(cfg config.LlamaServerConfig) *LlamaServerProvider {
	timeout := cfg.HTTPTimeout

	if timeout == 0 {
		timeout = 300 * time.Second
	}

	return &LlamaServerProvider{
		cfg:    cfg,
		client: &http.Client{Timeout: timeout},
	}
}

type LlamaServerStatus struct {
	Endpoint  string
	AutoStart bool
	Healthy   bool
	Running   bool
	PID       int
	StartedAt time.Time
}

func (p *LlamaServerProvider) Status() LlamaServerStatus {
	p.mu.Lock()
	processID := 0

	if p.cmd != nil && p.cmd.Process != nil {
		processID = p.cmd.Process.Pid
	}

	endpoint := p.cfg.Endpoint
	autoStart := p.cfg.AutoStart
	started := p.start
	p.mu.Unlock()

	return LlamaServerStatus{
		Endpoint:  endpoint,
		AutoStart: autoStart,
		Healthy:   p.isHealthy(),
		Running:   processID != 0,
		PID:       processID,
		StartedAt: started,
	}
}

func (p *LlamaServerProvider) Stop() {
	p.stopProcess()
}

func (p *LlamaServerProvider) CountTokens(text string) (int, error) {
	if err := p.ensureRunning(); err != nil {
		return estimateTokens(text), nil
	}

	endpoint := p.cfg.Endpoint
	endpointURL := endpoint + "/tokenize"
	requestBody, _ := json.Marshal(map[string]any{"content": text})
	httpResp, err := p.client.Post(endpointURL, "application/json", bytes.NewReader(requestBody))

	if err != nil {
		return estimateTokens(text), nil
	}
	defer httpResp.Body.Close()

	var payload map[string]any

	if err := json.NewDecoder(httpResp.Body).Decode(&payload); err != nil {
		return estimateTokens(text), nil
	}

	if tokens, ok := payload["tokens"].([]any); ok {
		return len(tokens), nil
	}

	if count, ok := payload["count"].(float64); ok {
		return int(count), nil
	}

	return estimateTokens(text), nil
}

func (p *LlamaServerProvider) GenerateChat(messages []core.Message, tools []core.ToolDef,
	tokenCb func(string) bool, reasoningCb func(string) bool, sampling *core.SamplingConfig, model string, useToolRole bool) (core.ChatResponse, error) {
	if err := p.ensureRunning(); err != nil {
		return core.ChatResponse{}, err
	}

	endpoint := p.cfg.Endpoint
	endpointURL := endpoint + "/v1/chat/completions"

	msgJSON := make([]map[string]any, 0, len(messages))

	for _, message := range messages {
		entry := map[string]any{"role": string(message.Role), "content": message.Content}

		if len(message.ToolCalls) > 0 {
			entry["tool_calls"] = toToolCalls(message.ToolCalls)
		}

		if message.ToolResult != nil {
			if useToolRole {
				entry["role"] = "tool"
				entry["content"] = message.ToolResult.Output

				if message.ToolResult.CallID != "" {
					entry["tool_call_id"] = message.ToolResult.CallID
				}
			} else {
				entry["role"] = "user"
				toolName := message.ToolResult.Name
				toolOutput := message.ToolResult.Output
				entry["content"] = "Tool: " + toolName + "\n\nResult:\n" + toolOutput
			}
		}

		msgJSON = append(msgJSON, entry)
	}

	toolJSON := make([]map[string]any, 0, len(tools))

	for _, t := range tools {
		toolJSON = append(toolJSON, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        t.Name,
				"description": t.Description,
				"parameters":  t.Parameters,
			},
		})
	}

	modelName := model
	if modelName == "" {
		modelName = "default"
	}
	modelName = strings.TrimSuffix(modelName, ".gguf")

	maxTokens := 4096
	if sampling != nil && sampling.MaxTokens != nil {
		maxTokens = *sampling.MaxTokens
	}

	payload := map[string]any{
		"model":      modelName,
		"messages":   msgJSON,
		"tools":      toolJSON,
		"max_tokens": maxTokens,
		"stream":     false,
	}

	if sampling != nil {
		if sampling.Temperature != nil {
			payload["temperature"] = *sampling.Temperature
		}
		if sampling.TopP != nil {
			payload["top_p"] = *sampling.TopP
		}
		if sampling.TopK != nil {
			payload["top_k"] = *sampling.TopK
		}
		if sampling.RepeatPenalty != nil {
			payload["repeat_penalty"] = *sampling.RepeatPenalty
		}
	}

	body, _ := json.Marshal(payload)
	httpResp, err := p.client.Post(endpointURL, "application/json", bytes.NewReader(body))

	if err != nil {
		return core.ChatResponse{}, err
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(httpResp.Body)

		if len(bodyBytes) > 0 {
			return core.ChatResponse{}, errors.New(httpResp.Status + ": " + strings.TrimSpace(string(bodyBytes)))
		}

		return core.ChatResponse{}, errors.New(httpResp.Status)
	}

	var responsePayload map[string]any

	if err := json.NewDecoder(httpResp.Body).Decode(&responsePayload); err != nil {
		return core.ChatResponse{}, err
	}

	choices, _ := responsePayload["choices"].([]any)

	if len(choices) == 0 {
		return core.ChatResponse{}, errors.New("no choices")
	}

	choice, _ := choices[0].(map[string]any)
	message, _ := choice["message"].(map[string]any)
	content, _ := message["content"].(string)
	reasoning, _ := message["reasoning_content"].(string)
	toolCalls := parseToolCalls(message)

	if tokenCb != nil && content != "" {
		if !tokenCb(content) {
			return core.ChatResponse{}, errors.New("cancelled")
		}
	}

	if reasoningCb != nil && reasoning != "" {
		if !reasoningCb(reasoning) {
			return core.ChatResponse{}, errors.New("cancelled")
		}
	}

	return core.ChatResponse{Content: content, Reasoning: reasoning, ToolCalls: toolCalls}, nil
}

func (p *LlamaServerProvider) ConcurrencyLimit() int {
	return 1
}

func (p *LlamaServerProvider) Start() error {
	if !p.cfg.AutoStart {
		return errors.New("auto_start disabled")
	}

	if p.cfg.ModelDir == "" {
		return errors.New("model_dir required")
	}

	if _, err := os.Stat(p.cfg.ModelDir); err != nil {
		return err
	}

	p.stopProcess()
	return p.spawnProcess()
}

func (p *LlamaServerProvider) ensureRunning() error {
	if p.isHealthy() {
		return nil
	}

	if !p.cfg.AutoStart {
		return errors.New("llama-server not reachable")
	}

	if err := p.Start(); err != nil {
		return err
	}

	return p.waitReady()
}

func (p *LlamaServerProvider) isHealthy() bool {
	endpoint := p.cfg.Endpoint
	url := endpoint + "/v1/models"
	resp, err := p.client.Get(url)

	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

func (p *LlamaServerProvider) spawnProcess() error {
	endpoint := p.cfg.Endpoint
	parsed, err := url.Parse(endpoint)

	if err != nil {
		return err
	}

	host := parsed.Hostname()

	if host == "" {
		host = "127.0.0.1"
	}

	port := parsed.Port()

	if port == "" {
		port = "8080"
	}

	bin := p.cfg.BinPath

	if bin == "" {
		bin = "llama-server"
	}

	args := []string{
		"--host", host,
		"--port", port,
		"--ctx-size", intToString(p.cfg.ContextSize),
		"--n-gpu-layers", intToString(p.cfg.GPULayers),
		"--models-dir", p.cfg.ModelDir,
		"--reasoning-format", "deepseek",
	}

	cmd := exec.Command(bin, args...)
	if p.cfg.InheritStdio {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	cmd.Dir = p.cfg.ModelDir

	if err := cmd.Start(); err != nil {
		return err
	}

	p.mu.Lock()
	p.cmd = cmd
	p.start = time.Now()
	p.mu.Unlock()

	return nil
}

func (p *LlamaServerProvider) stopProcess() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cmd == nil || p.cmd.Process == nil {
		return
	}

	_ = p.cmd.Process.Kill()
	p.cmd = nil
}

func (p *LlamaServerProvider) waitReady() error {
	wait := p.cfg.StartupWait

	if wait == 0 {
		wait = 10 * time.Second
	}

	deadline := time.Now().Add(wait)

	for time.Now().Before(deadline) {
		if p.isHealthy() {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}

	return errors.New("llama-server did not become ready")
}

func estimateTokens(text string) int {
	return len(text) / 4
}

func intToString(v int) string {
	return strconv.Itoa(v)
}

func toToolCalls(calls []core.ToolCall) []map[string]any {
	var toolCalls []map[string]any

	for _, call := range calls {
		argsJSON, _ := json.Marshal(call.Arguments)
		toolCalls = append(toolCalls, map[string]any{
			"id":   call.ID,
			"type": "function",
			"function": map[string]any{
				"name":      call.Name,
				"arguments": string(argsJSON),
			},
		})
	}

	return toolCalls
}

func parseToolCalls(message map[string]any) []core.ToolCall {
	rawCalls, ok := message["tool_calls"].([]any)

	if !ok {
		return nil
	}

	var toolCalls []core.ToolCall

	for _, rawCall := range rawCalls {
		rawEntry, _ := rawCall.(map[string]any)
		callID, _ := rawEntry["id"].(string)
		functionEntry, _ := rawEntry["function"].(map[string]any)
		functionName, _ := functionEntry["name"].(string)
		arguments := map[string]any{}

		switch v := functionEntry["arguments"].(type) {
		case string:
			if v != "" {
				_ = json.Unmarshal([]byte(v), &arguments)
			}
		case map[string]any:
			arguments = v
		}

		toolCalls = append(toolCalls, core.ToolCall{ID: callID, Name: functionName, Arguments: arguments})
	}

	return toolCalls
}
