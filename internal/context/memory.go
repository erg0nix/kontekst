package context

import (
	"strings"

	"github.com/erg0nix/kontekst/internal/core"
)

type InMemoryContext struct {
	Messages          []core.Message
	MaxTokens         int
	SystemTemplate    string
	UserTemplate      string
	AgentSystemPrompt string
}

func (contextWindow *InMemoryContext) AddMessage(msg core.Message) error {
	contextWindow.Messages = append(contextWindow.Messages, msg)

	return nil
}

func (contextWindow *InMemoryContext) BuildContext(_ func(string) (int, error)) ([]core.Message, error) {
	systemContent := contextWindow.SystemTemplate
	if contextWindow.AgentSystemPrompt != "" {
		systemContent = systemContent + "\n\n---\n\n" + contextWindow.AgentSystemPrompt
	}

	systemMessage := core.Message{Role: core.RoleSystem, Content: systemContent}
	out := []core.Message{systemMessage}
	out = append(out, contextWindow.Messages...)

	return out, nil
}

func (contextWindow *InMemoryContext) SetAgentSystemPrompt(prompt string) {
	contextWindow.AgentSystemPrompt = prompt
}

func (contextWindow *InMemoryContext) RenderUserMessage(prompt string) (string, error) {
	if contextWindow.UserTemplate == "" {
		return prompt, nil
	}

	return strings.ReplaceAll(contextWindow.UserTemplate, "{{ user_message }}", prompt), nil
}

func (contextWindow *InMemoryContext) AddToolResult(result core.ToolResult) error {
	msg := core.Message{Role: core.RoleTool, Content: result.Output, ToolResult: &result}
	contextWindow.Messages = append(contextWindow.Messages, msg)

	return nil
}

type InMemoryContextService struct{}

func (service *InMemoryContextService) NewWindow(_ core.SessionID) (ContextWindow, error) {
	return &InMemoryContext{
		SystemTemplate: "You are a helpful assistant.",
		UserTemplate:   "{{ user_message }}",
	}, nil
}
