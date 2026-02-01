package skills

import "fmt"

type Skill struct {
	Name                   string
	Description            string
	Content                string
	Path                   string
	DisableModelInvocation bool
	UserInvocable          bool
}

func (s *Skill) FormatContent(renderedContent string) string {
	return fmt.Sprintf("[Skill: %s]\nBase path: %s\n\n%s", s.Name, s.Path, renderedContent)
}
