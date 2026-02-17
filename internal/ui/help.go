package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	styleHelpBorder = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(colorAccent).
			Background(colorBg).
			Padding(1, 2)

	styleHelpTitle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	styleHelpKey = lipgloss.NewStyle().
			Foreground(colorYellow).
			Bold(true)

	styleHelpDesc = lipgloss.NewStyle().
			Foreground(colorFg)

	styleHelpSection = lipgloss.NewStyle().
				Foreground(colorCyan).
				Bold(true)
)

func renderHelp(width, height int) string {
	kv := func(key, desc string) string {
		return styleHelpKey.Render(key) + styleHelpDesc.Render("  "+desc)
	}

	// Left column: Global + Process Table
	var leftCol []string
	leftCol = append(leftCol, styleHelpSection.Render("Navigation"))
	leftCol = append(leftCol, kv("j/k ↑↓  ", "move up/down"))
	leftCol = append(leftCol, kv("PgUp/Dn ", "page up/down"))
	leftCol = append(leftCol, kv("g/G     ", "first/last"))
	leftCol = append(leftCol, "")
	leftCol = append(leftCol, styleHelpSection.Render("Process Table"))
	leftCol = append(leftCol, kv("enter   ", "open detail"))
	leftCol = append(leftCol, kv("s       ", "cycle sort"))
	leftCol = append(leftCol, kv("/       ", "search/filter"))
	leftCol = append(leftCol, kv("h       ", "remote hosts"))
	leftCol = append(leftCol, kv("l       ", "listen ports"))
	leftCol = append(leftCol, kv("K       ", "kill process"))
	leftCol = append(leftCol, kv("D       ", "group view"))

	// Right column: Detail + Global
	var rightCol []string
	rightCol = append(rightCol, styleHelpSection.Render("Process Detail"))
	rightCol = append(rightCol, kv("d       ", "toggle DNS"))
	rightCol = append(rightCol, kv("K       ", "kill process"))
	rightCol = append(rightCol, kv("esc     ", "back to table"))
	rightCol = append(rightCol, "")
	rightCol = append(rightCol, styleHelpSection.Render("Global"))
	rightCol = append(rightCol, kv("i / tab ", "cycle interface"))
	rightCol = append(rightCol, kv("+ / -   ", "refresh speed"))
	rightCol = append(rightCol, kv("space   ", "pause/resume"))
	rightCol = append(rightCol, kv("← / →   ", "playback speed"))
	rightCol = append(rightCol, kv("?       ", "toggle help"))
	rightCol = append(rightCol, kv("q       ", "quit"))

	left := strings.Join(leftCol, "\n")
	right := strings.Join(rightCol, "\n")

	columns := lipgloss.JoinHorizontal(lipgloss.Top,
		left, "    ", right,
	)

	title := styleHelpTitle.Render("  Keyboard Shortcuts")

	content := title + "\n\n" + columns

	box := styleHelpBorder.Render(content)

	// Center the box
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}
