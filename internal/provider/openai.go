package provider

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/erg0nix/kontekst/internal/config"
	"github.com/erg0nix/kontekst/internal/core"
)

// OpenAIConfig holds connection settings for an OpenAI-compatible API endpoint.
type OpenAIConfig struct {
	Endpoint    string
	HTTPTimeout time.Duration
}

// OpenAIProvider implements Provider using an OpenAI-compatible HTTP API.
type OpenAIProvider struct {
	endpoint      string
	client        *http.Client
	requestLogger *RequestLogger
	validateRoles bool
}

// NewOpenAIProvider creates an OpenAIProvider with the given endpoint config and optional debug logging.
func NewOpenAIProvider(cfg OpenAIConfig, debugCfg config.DebugConfig) *OpenAIProvider {
	timeout := cfg.HTTPTimeout
	if timeout == 0 {
		timeout = 300 * time.Second
	}

	provider := &OpenAIProvider{
		endpoint: cfg.Endpoint,
		client:   &http.Client{Timeout: timeout},
	}

	if debugCfg.LogRequests || debugCfg.LogResponses {
		provider.requestLogger = NewRequestLogger(
			debugCfg.LogDirectory,
			debugCfg.LogRequests,
			debugCfg.LogResponses,
			slog.Default(),
		)
	}

	provider.validateRoles = debugCfg.ValidateRoles

	return provider
}

// GenerateChat sends a chat completion request to the OpenAI-compatible endpoint and returns the parsed response.
func (p *OpenAIProvider) GenerateChat(
	messages []core.Message,
	tools []core.ToolDef,
	sampling *core.SamplingConfig,
	model string,
	useToolRole bool,
) (Response, error) {
	requestID := core.NewRequestID()

	messages = normalizeMessages(messages, useToolRole)

	if p.validateRoles {
		if err := validateRoleAlternation(messages, useToolRole); err != nil {
			if p.requestLogger != nil {
				p.requestLogger.LogError(requestID, 0, []byte(err.Error()), messages, nil)
			}
			return Response{}, fmt.Errorf("role validation failed (request_id=%s): %w", requestID, err)
		}
	}

	endpointURL := p.endpoint + "/v1/chat/completions"

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

	body, err := json.Marshal(payload)
	if err != nil {
		return Response{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	if p.requestLogger != nil {
		p.requestLogger.LogRequest(requestID, messages, tools, sampling, payload)
	}

	startTime := time.Now()
	httpResp, err := p.client.Post(endpointURL, "application/json", bytes.NewReader(body))
	duration := time.Since(startTime)

	if err != nil {
		if p.requestLogger != nil {
			p.requestLogger.LogError(requestID, 0, []byte(err.Error()), messages, payload)
		}
		return Response{}, fmt.Errorf("provider request failed (request_id=%s): %w", requestID, err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(httpResp.Body)

		if p.requestLogger != nil {
			p.requestLogger.LogError(requestID, httpResp.StatusCode, bodyBytes, messages, payload)
		}

		if len(bodyBytes) > 0 {
			return Response{}, fmt.Errorf("provider error (request_id=%s): %s: %s",
				requestID, httpResp.Status, strings.TrimSpace(string(bodyBytes)))
		}

		return Response{}, fmt.Errorf("provider error (request_id=%s): %s", requestID, httpResp.Status)
	}

	var responsePayload map[string]any
	if err := json.NewDecoder(httpResp.Body).Decode(&responsePayload); err != nil {
		return Response{}, fmt.Errorf("decode response: %w", err)
	}

	response, err := parseResponsePayload(responsePayload)
	if err != nil {
		return Response{}, fmt.Errorf("provider response parse failed (request_id=%s): %w", requestID, err)
	}

	if p.requestLogger != nil {
		p.requestLogger.LogResponse(requestID, response, duration)
	}

	return response, nil
}

// CountTokens returns the token count for the given text, falling back to estimation if the endpoint is unavailable.
func (p *OpenAIProvider) CountTokens(text string) (int, error) {
	endpointURL := p.endpoint + "/tokenize"
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

func estimateTokens(text string) int {
	return len(text) / 4
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

func parseResponsePayload(payload map[string]any) (Response, error) {
	choices, ok := payload["choices"].([]any)
	if !ok || len(choices) == 0 {
		return Response{}, errors.New("no choices in response")
	}

	choice, ok := choices[0].(map[string]any)
	if !ok {
		return Response{}, errors.New("malformed choice in response")
	}

	message, ok := choice["message"].(map[string]any)
	if !ok {
		return Response{}, errors.New("malformed message in response")
	}

	content, _ := message["content"].(string)
	reasoning, _ := message["reasoning_content"].(string)

	return Response{
		Content:   content,
		Reasoning: reasoning,
		ToolCalls: parseToolCalls(message),
		Usage:     parseUsage(payload),
	}, nil
}

func parseToolCalls(message map[string]any) []core.ToolCall {
	rawCalls, ok := message["tool_calls"].([]any)
	if !ok {
		return nil
	}

	var toolCalls []core.ToolCall
	for _, rawCall := range rawCalls {
		rawEntry, ok := rawCall.(map[string]any)
		if !ok {
			continue
		}

		callID, _ := rawEntry["id"].(string)

		functionEntry, ok := rawEntry["function"].(map[string]any)
		if !ok {
			continue
		}

		functionName, _ := functionEntry["name"].(string)
		if functionName == "" {
			continue
		}

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

func parseUsage(response map[string]any) *Usage {
	usageMap, ok := response["usage"].(map[string]any)
	if !ok {
		return nil
	}

	return &Usage{
		PromptTokens:     intFromAny(usageMap["prompt_tokens"]),
		CompletionTokens: intFromAny(usageMap["completion_tokens"]),
		TotalTokens:      intFromAny(usageMap["total_tokens"]),
	}
}

func intFromAny(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	default:
		return 0
	}
}
