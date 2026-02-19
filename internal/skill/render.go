package skill

import (
	"strings"
)

func (s *Skill) Render(arguments string) (string, error) {
	content := s.Content

	content = strings.ReplaceAll(content, "$ARGUMENTS", arguments)

	args := parseArguments(arguments)
	for i, arg := range args {
		placeholder := "$" + string(rune('0'+i))
		content = strings.ReplaceAll(content, placeholder, arg)
	}

	return content, nil
}

func parseArguments(arguments string) []string {
	if arguments == "" {
		return nil
	}

	var result []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)

	for _, r := range arguments {
		switch {
		case !inQuote && (r == '"' || r == '\''):
			inQuote = true
			quoteChar = r
		case inQuote && r == quoteChar:
			inQuote = false
			quoteChar = 0
		case !inQuote && r == ' ':
			if current.Len() > 0 {
				result = append(result, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}

	if current.Len() > 0 {
		result = append(result, current.String())
	}

	return result
}
