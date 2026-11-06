package main

import "charm.land/lipgloss/v2"

var (
	brandStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#c4b5fd"))
	activeTabStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#f0abfc")).Underline(true)
	mutedStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#a1a1aa"))
)
