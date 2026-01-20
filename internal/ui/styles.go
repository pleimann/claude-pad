package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// Color palette
var (
	ColorPrimary   = lipgloss.Color("#7C3AED") // Purple
	ColorSecondary = lipgloss.Color("#6366F1") // Indigo
	ColorSuccess   = lipgloss.Color("#10B981") // Green
	ColorWarning   = lipgloss.Color("#F59E0B") // Amber
	ColorError     = lipgloss.Color("#EF4444") // Red
	ColorMuted     = lipgloss.Color("#6B7280") // Gray
	ColorSubtle    = lipgloss.Color("#9CA3AF") // Light gray
)

// Text styles
var (
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(ColorSuccess)

	WarningStyle = lipgloss.NewStyle().
			Foreground(ColorWarning)

	ErrorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorError)

	MutedStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	SubtleStyle = lipgloss.NewStyle().
			Foreground(ColorSubtle)

	BoldStyle = lipgloss.NewStyle().
			Bold(true)

	CodeStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Background(lipgloss.Color("#1F2937")).
			Padding(0, 1)
)

// Box styles
var (
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorMuted).
			Padding(0, 1)

	HighlightBoxStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorPrimary).
				Padding(0, 1)
)

// Device listing styles
var (
	DeviceIDStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Bold(true)

	DeviceNameStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F9FAFB"))

	DeviceManufacturerStyle = lipgloss.NewStyle().
				Foreground(ColorMuted)

	DeviceIndexStyle = lipgloss.NewStyle().
				Foreground(ColorPrimary).
				Bold(true)
)

// Helper functions

// Title renders a styled title
func Title(text string) string {
	return TitleStyle.Render(text)
}

// Subtitle renders a styled subtitle
func Subtitle(text string) string {
	return SubtitleStyle.Render(text)
}

// Success renders success text with a checkmark
func Success(text string) string {
	return SuccessStyle.Render("✓ " + text)
}

// Warning renders warning text
func Warning(text string) string {
	return WarningStyle.Render("⚠ " + text)
}

// Error renders error text
func Error(text string) string {
	return ErrorStyle.Render("✗ " + text)
}

// Muted renders muted/dimmed text
func Muted(text string) string {
	return MutedStyle.Render(text)
}

// Code renders inline code
func Code(text string) string {
	return CodeStyle.Render(text)
}

// Bold renders bold text
func Bold(text string) string {
	return BoldStyle.Render(text)
}
