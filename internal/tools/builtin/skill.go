package builtin

import (
	"context"
	"fmt"
	"strings"

	"github.com/erg0nix/kontekst/internal/core"
	"github.com/erg0nix/kontekst/internal/skills"
	"github.com/erg0nix/kontekst/internal/tools"
)

type skillContextKey struct{}

type SkillCallbacks struct {
	ContextInjector func(msg core.Message) error
}

func WithSkillCallbacks(ctx context.Context, callbacks *SkillCallbacks) context.Context {
	return context.WithValue(ctx, skillContextKey{}, callbacks)
}

func GetSkillCallbacks(ctx context.Context) *SkillCallbacks {
	val := ctx.Value(skillContextKey{})
	if val == nil {
		return nil
	}
	return val.(*SkillCallbacks)
}

type SkillTool struct {
	Registry *skills.Registry
}

func (tool *SkillTool) Name() string { return "skill" }

func (tool *SkillTool) Description() string {
	var sb strings.Builder
	sb.WriteString("Invokes a skill by name. Skills provide specialized workflows and instructions.\n\n")

	modelSkills := tool.Registry.ModelInvocableSkills()
	if len(modelSkills) > 0 {
		sb.WriteString("<available_skills>\n")
		for _, s := range modelSkills {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", s.Name, s.Description))
		}
		sb.WriteString("</available_skills>\n\n")
		sb.WriteString("When a task matches a skill's description, invoke it to get detailed instructions.")
	} else {
		sb.WriteString("No skills are currently available.")
	}

	return sb.String()
}

func (tool *SkillTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name":      map[string]any{"type": "string", "description": "Skill name from available_skills list"},
			"arguments": map[string]any{"type": "string", "description": "Arguments to pass to the skill (optional)"},
		},
		"required": []string{"name"},
	}
}

func (tool *SkillTool) RequiresApproval() bool { return false }

func (tool *SkillTool) Execute(args map[string]any, ctx context.Context) (string, error) {
	name, _ := args["name"].(string)
	arguments, _ := args["arguments"].(string)

	if name == "" {
		return "", fmt.Errorf("skill name is required")
	}

	skill, found := tool.Registry.Get(name)
	if !found {
		return "", fmt.Errorf("skill not found: %s", name)
	}

	if skill.DisableModelInvocation {
		return "", fmt.Errorf("skill '%s' can only be invoked by user with /%s", name, name)
	}

	rendered, err := skill.Render(arguments)
	if err != nil {
		return "", fmt.Errorf("failed to render skill: %w", err)
	}

	callbacks := GetSkillCallbacks(ctx)
	if callbacks == nil {
		return "", fmt.Errorf("skill execution not supported in this context")
	}

	msg := core.Message{
		Role: core.RoleUser,
		Content: fmt.Sprintf("[Skill: %s]\nBase path: %s\n\n%s",
			skill.Name, skill.Path, rendered),
	}
	if err := callbacks.ContextInjector(msg); err != nil {
		return "", fmt.Errorf("failed to inject skill content: %w", err)
	}

	return fmt.Sprintf("Skill '%s' loaded. Follow the instructions in the message above.", name), nil
}

func RegisterSkill(registry *tools.Registry, skillsRegistry *skills.Registry) {
	registry.Add(&SkillTool{Registry: skillsRegistry})
}
