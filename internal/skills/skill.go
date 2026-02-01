package skills

type Skill struct {
	Name                   string
	Description            string
	Content                string
	Path                   string
	DisableModelInvocation bool
	UserInvocable          bool
}
