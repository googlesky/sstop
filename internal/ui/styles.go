package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// Tokyo Night inspired color palette
var (
	colorBg        = lipgloss.Color("#1a1b26")
	colorFg        = lipgloss.Color("#a9b1d6")
	colorFgDim     = lipgloss.Color("#565f89")
	colorAccent    = lipgloss.Color("#7aa2f7") // blue
	colorGreen     = lipgloss.Color("#9ece6a")
	colorRed       = lipgloss.Color("#f7768e")
	colorYellow    = lipgloss.Color("#e0af68")
	colorCyan      = lipgloss.Color("#7dcfff")
	colorMagenta   = lipgloss.Color("#bb9af7")
	colorBorder    = lipgloss.Color("#3b4261")
	colorSelection = lipgloss.Color("#283457")
	colorZebraRow  = lipgloss.Color("#1e2030") // subtle alternating row bg
)

var (
	styleHeaderValue = lipgloss.NewStyle().
				Foreground(colorFg)

	styleHeaderUp = lipgloss.NewStyle().
			Foreground(colorGreen).
			Bold(true)

	styleHeaderDown = lipgloss.NewStyle().
			Foreground(colorRed).
			Bold(true)

	styleTableHeader = lipgloss.NewStyle().
				Foreground(colorAccent).
				Bold(true)

	styleTableRow = lipgloss.NewStyle().
			Foreground(colorFg)

	styleTableRowSelected = lipgloss.NewStyle().
				Background(colorSelection).
				Foreground(colorFg)

	styleUpRate = lipgloss.NewStyle().
			Foreground(colorGreen)

	styleDownRate = lipgloss.NewStyle().
			Foreground(colorRed)

	stylePID = lipgloss.NewStyle().
			Foreground(colorFgDim)

	styleProcessName = lipgloss.NewStyle().
				Foreground(colorFg).
				Bold(true)

	styleConnCount = lipgloss.NewStyle().
			Foreground(colorCyan)

	styleListenCount = lipgloss.NewStyle().
				Foreground(colorMagenta)

	styleSortIndicator = lipgloss.NewStyle().
				Foreground(colorYellow).
				Bold(true)

	styleFooter = lipgloss.NewStyle().
			Foreground(colorFgDim)

	styleFooterKey = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	styleBorder = lipgloss.NewStyle().
			Foreground(colorBorder)

	styleTitle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	styleDetailLabel = lipgloss.NewStyle().
				Foreground(colorFgDim)

	styleStateEstablished = lipgloss.NewStyle().Foreground(colorGreen)
	styleStateListen      = lipgloss.NewStyle().Foreground(colorCyan)
	styleStateTimeWait    = lipgloss.NewStyle().Foreground(colorFgDim)
	styleStateClosing     = lipgloss.NewStyle().Foreground(colorYellow)
	styleStateOther       = lipgloss.NewStyle().Foreground(colorFg)

	styleSearchPrompt = lipgloss.NewStyle().
				Foreground(colorYellow).
				Bold(true)

	styleSparkline = lipgloss.NewStyle().
			Foreground(colorBorder)

	styleSparklineActive = lipgloss.NewStyle().
				Foreground(colorCyan)

	stylePaused = lipgloss.NewStyle().
			Foreground(colorYellow).
			Bold(true)

	styleZebraRow = lipgloss.NewStyle().
			Background(colorZebraRow)

	styleAlertTag = lipgloss.NewStyle().
			Foreground(colorRed).
			Bold(true)
)

// rateColorIntensity returns a lipgloss.Color that interpolates between dim and vivid
// based on rate/maxRate ratio. baseH is the hue (green=96, red=354).
func rateColorIntensity(rate, maxRate float64, baseH float64) lipgloss.Color {
	if maxRate <= 0 || rate <= 0 {
		return colorFgDim
	}
	t := clamp01(rate / maxRate)
	// Dim: low saturation, low lightness → Vivid: high saturation, high lightness
	s := lerpValue(0.2, 0.85, t)
	l := lerpValue(0.35, 0.65, t)
	r, g, b := hslToRGB(baseH, s, l)
	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b))
}

// Green hue for upload, red hue for download
const (
	hueGreen = 96.0  // matches #9ece6a
	hueRed   = 354.0 // matches #f7768e
)

// barStyle returns a lipgloss.Style for bandwidth bar with rate-based color intensity.
func barStyleUp(rate, maxRate float64) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(rateColorIntensity(rate, maxRate, hueGreen))
}

func barStyleDown(rate, maxRate float64) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(rateColorIntensity(rate, maxRate, hueRed))
}
