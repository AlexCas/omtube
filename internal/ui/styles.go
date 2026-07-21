package ui

import "github.com/charmbracelet/lipgloss"

type styles struct {
	title    lipgloss.Style
	panel    lipgloss.Style
	heading  lipgloss.Style
	selected lipgloss.Style
	current  lipgloss.Style
	dim      lipgloss.Style
	help     lipgloss.Style
	errorMsg lipgloss.Style
	viz      lipgloss.Style
}

func defaultStyles() styles {
	return styles{
		title: lipgloss.NewStyle().Bold(true).
			Foreground(lipgloss.Color("#e0aaff")).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#e0aaff")).
			Padding(0, 1),
		panel: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#e0aaff")).
			Padding(0, 1),
		heading:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#e0aaff")),
		selected: lipgloss.NewStyle().Foreground(lipgloss.Color("#00f5d4")).Bold(true),
		current:  lipgloss.NewStyle().Foreground(lipgloss.Color("#00f5d4")).Bold(true),
		dim:      lipgloss.NewStyle().Foreground(lipgloss.Color("#a0a0a0")),
		help:     lipgloss.NewStyle().Foreground(lipgloss.Color("#a0a0a0")),
		errorMsg: lipgloss.NewStyle().Foreground(lipgloss.Color("#e0aaff")).Bold(true),
		viz:      lipgloss.NewStyle().Foreground(lipgloss.Color("#e0aaff")),
	}
}
