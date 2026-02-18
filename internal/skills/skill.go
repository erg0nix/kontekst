package skills

import "fmt"

// Skill represents a reusable prompt template that can be invoked by models or users.
type Skill struct {
	Name                   string
	Description            string
	Content                string
	Path                   string
	DisableModelInvocation bool
	UserInvocable          bool
}

// FormatContent wraps the rendered skill content with the skill's name and base path header.
func (s *Skill) FormatContent(renderedContent string) string {
	return fmt.Sprintf("[Skill: %s]\nBase path: %s\n\n%s", s.Name, s.Path, renderedContent)
}
