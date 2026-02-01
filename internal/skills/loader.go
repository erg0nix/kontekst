package skills

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

type skillFrontmatter struct {
	Name                   string `toml:"name"`
	Description            string `toml:"description"`
	DisableModelInvocation bool   `toml:"disable_model_invocation"`
	UserInvocable          *bool  `toml:"user_invocable"`
}

func loadSkillFile(path string) (*Skill, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	content := string(data)
	frontmatter, body := parseFrontmatter(content)

	var fm skillFrontmatter
	if frontmatter != "" {
		if err := toml.Unmarshal([]byte(frontmatter), &fm); err != nil {
			return nil, err
		}
	}

	name := fm.Name
	if name == "" {
		name = deriveNameFromPath(path)
	}

	userInvocable := true
	if fm.UserInvocable != nil {
		userInvocable = *fm.UserInvocable
	}

	return &Skill{
		Name:                   name,
		Description:            fm.Description,
		Content:                strings.TrimSpace(body),
		Path:                   filepath.Dir(path),
		DisableModelInvocation: fm.DisableModelInvocation,
		UserInvocable:          userInvocable,
	}, nil
}

func parseFrontmatter(content string) (frontmatter, body string) {
	content = strings.TrimSpace(content)

	if !strings.HasPrefix(content, "+++") {
		return "", content
	}

	rest := strings.TrimPrefix(content, "+++")
	endIndex := strings.Index(rest, "\n+++")
	if endIndex == -1 {
		return "", content
	}

	frontmatter = strings.TrimSpace(rest[:endIndex])
	body = strings.TrimPrefix(rest[endIndex:], "\n+++")

	return frontmatter, body
}

func deriveNameFromPath(path string) string {
	dir := filepath.Dir(path)
	base := filepath.Base(path)

	if base == "SKILL.md" {
		return filepath.Base(dir)
	}

	name := strings.TrimSuffix(base, filepath.Ext(base))
	return name
}
