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
}

func defaultStyles() styles {
	return styles{
		title: lipgloss.NewStyle().Bold(true).
			Foreground(lipgloss.Color("213")).Padding(0, 1),
		panel: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("60")).
			Padding(0, 1),
		heading:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("117")),
		selected: lipgloss.NewStyle().Foreground(lipgloss.Color("213")).Bold(true),
		current:  lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true),
		dim:      lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
		help:     lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
		errorMsg: lipgloss.NewStyle().Foreground(lipgloss.Color("203")),
	}
}
