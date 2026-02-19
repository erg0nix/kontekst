package cli

import (
	"image/color"

	lipgloss "github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/lipgloss/v2/table"
)

var (
	colorPrimary = lipgloss.Color("#7C71F9")
	colorSuccess = lipgloss.Color("#34D399")
	colorError   = lipgloss.Color("#F87171")
	colorWarning = lipgloss.Color("#FBBF24")
	colorDim     = lipgloss.Color("#6B7280")
	colorAccent  = lipgloss.Color("#60A5FA")
)

var (
	styleDim     = lipgloss.NewStyle().Foreground(colorDim)
	styleError   = lipgloss.NewStyle().Foreground(colorError)
	styleSuccess = lipgloss.NewStyle().Foreground(colorSuccess)
	styleWarning = lipgloss.NewStyle().Foreground(colorWarning)

	styleToolName = lipgloss.NewStyle().Bold(true).Foreground(colorAccent)
	styleToolArgs = styleDim

	styleReasoning = lipgloss.NewStyle().Faint(true).Italic(true)

	stylePromptAction = lipgloss.NewStyle().Bold(true).Foreground(colorWarning)
	stylePromptHint   = styleDim

	styleTableHeader = lipgloss.NewStyle().Bold(true).Foreground(colorPrimary)

	styleActive = lipgloss.NewStyle().Bold(true).Foreground(colorSuccess)
	stylePID    = lipgloss.NewStyle().Foreground(colorAccent)
)

var toolKindColors = map[string]color.Color{
	"read":    colorSuccess,
	"edit":    colorWarning,
	"delete":  colorError,
	"execute": colorAccent,
	"search":  colorDim,
	"fetch":   colorDim,
}

func toolKindStyle(kind string) lipgloss.Style {
	if c, ok := toolKindColors[kind]; ok {
		return lipgloss.NewStyle().Foreground(c)
	}
	return styleDim
}

var toolKindLabels = map[string]string{
	"read":    "read",
	"edit":    "edit",
	"delete":  "del",
	"execute": "exec",
	"search":  "search",
	"fetch":   "fetch",
}

func toolKindLabel(kind string) string {
	if label, ok := toolKindLabels[kind]; ok {
		return label
	}
	return "tool"
}

func newTable(headers ...string) *table.Table {
	return table.New().
		Headers(headers...).
		BorderTop(false).
		BorderBottom(false).
		BorderLeft(false).
		BorderRight(false).
		BorderColumn(false).
		BorderHeader(true).
		Border(lipgloss.NormalBorder()).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return styleTableHeader
			}
			return lipgloss.NewStyle().PaddingRight(2)
		})
}

func styledError(msg string, hints ...string) string {
	out := styleError.Render(msg)
	for _, h := range hints {
		out += "\n  " + styleDim.Render(h)
	}
	return out
}
