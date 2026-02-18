package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Registry loads and stores skills from a directory, providing thread-safe access by name.
type Registry struct {
	skillsDir string
	skills    map[string]*Skill
	mu        sync.RWMutex
}

// NewRegistry creates a Registry that loads skills from the given directory.
func NewRegistry(skillsDir string) *Registry {
	return &Registry{
		skillsDir: skillsDir,
		skills:    make(map[string]*Skill),
	}
}

// Load discovers and parses all skill files from the registry's directory.
func (r *Registry) Load() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.skills = make(map[string]*Skill)

	if _, err := os.Stat(r.skillsDir); os.IsNotExist(err) {
		return nil
	}

	entries, err := os.ReadDir(r.skillsDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		name := entry.Name()

		if entry.IsDir() {
			skillPath := filepath.Join(r.skillsDir, name, "SKILL.md")
			if _, err := os.Stat(skillPath); err == nil {
				skill, err := loadSkillFile(skillPath)
				if err != nil {
					continue
				}
				r.skills[skill.Name] = skill
			}
			continue
		}

		if strings.HasSuffix(name, ".md") {
			skillPath := filepath.Join(r.skillsDir, name)
			skill, err := loadSkillFile(skillPath)
			if err != nil {
				continue
			}
			r.skills[skill.Name] = skill
		}
	}

	return nil
}

// Get returns the skill with the given name, or false if not found.
func (r *Registry) Get(name string) (*Skill, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	skill, ok := r.skills[name]
	return skill, ok
}

// ModelInvocableSkills returns all skills that have not disabled model invocation.
func (r *Registry) ModelInvocableSkills() []*Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*Skill
	for _, skill := range r.skills {
		if !skill.DisableModelInvocation {
			result = append(result, skill)
		}
	}
	return result
}

// Summaries returns a formatted string listing all model-invocable skills with their descriptions.
func (r *Registry) Summaries() string {
	skills := r.ModelInvocableSkills()
	if len(skills) == 0 {
		return ""
	}

	var sb strings.Builder
	for _, s := range skills {
		sb.WriteString(fmt.Sprintf("- %s: %s\n", s.Name, s.Description))
	}
	return sb.String()
}
