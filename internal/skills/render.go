package skills

import (
	"regexp"
	"strings"
)

type ShellCommand struct {
	Command     string
	Placeholder string
}

func (s *Skill) Render(arguments string) (string, []ShellCommand, error) {
	content := s.Content

	content = strings.ReplaceAll(content, "$ARGUMENTS", arguments)

	args := parseArguments(arguments)
	for i, arg := range args {
		placeholder := "$" + string(rune('0'+i))
		content = strings.ReplaceAll(content, placeholder, arg)
	}

	shellCommands := extractShellCommands(content)

	return content, shellCommands, nil
}

func (s *Skill) RenderWithShellOutput(arguments string, shellOutputs map[string]string) (string, error) {
	content, _, err := s.Render(arguments)
	if err != nil {
		return "", err
	}

	for placeholder, output := range shellOutputs {
		content = strings.ReplaceAll(content, placeholder, output)
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

var shellCommandPattern = regexp.MustCompile("!`([^`]+)`")

func extractShellCommands(content string) []ShellCommand {
	matches := shellCommandPattern.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return nil
	}

	var commands []ShellCommand
	seen := make(map[string]bool)

	for _, match := range matches {
		placeholder := match[0]
		command := match[1]

		if seen[placeholder] {
			continue
		}
		seen[placeholder] = true

		commands = append(commands, ShellCommand{
			Command:     command,
			Placeholder: placeholder,
		})
	}

	return commands
}

func HasShellPreprocessing(content string) bool {
	return shellCommandPattern.MatchString(content)
}
