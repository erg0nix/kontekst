package main

import (
	"fmt"
	"image/color"

	lipgloss "github.com/charmbracelet/lipgloss/v2"
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
	styleDim = lipgloss.NewStyle().Foreground(colorDim)
	styleError   = lipgloss.NewStyle().Foreground(colorError)
	styleSuccess = lipgloss.NewStyle().Foreground(colorSuccess)
	styleWarning = lipgloss.NewStyle().Foreground(colorWarning)

	styleLabel = styleDim
	styleValue = lipgloss.NewStyle()

	styleToolName = lipgloss.NewStyle().Bold(true).Foreground(colorAccent)
	styleToolArgs = styleDim

	styleReasoning = lipgloss.NewStyle().Faint(true).Italic(true)

	stylePromptAction = lipgloss.NewStyle().Bold(true).Foreground(colorWarning)
	stylePromptHint   = styleDim

	styleTableHeader = lipgloss.NewStyle().Bold(true).Foreground(colorPrimary)

	styleActive     = lipgloss.NewStyle().Bold(true).Foreground(colorSuccess)
	stylePID        = lipgloss.NewStyle().Foreground(colorAccent)
	styleServerName = lipgloss.NewStyle().Bold(true).Foreground(colorPrimary)
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

func kvLine(key, value string) string {
	return fmt.Sprintf("  %s %s", styleLabel.Render(key+":"), styleValue.Render(value))
}

func styledError(msg string, hints ...string) string {
	out := styleError.Render(msg)
	for _, h := range hints {
		out += "\n  " + styleDim.Render(h)
	}
	return out
}
