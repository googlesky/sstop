package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/googlesky/sstop/internal/model"
)

func renderHeader(snap model.Snapshot, width int, paused bool, activeIface string) string {
	title := styleTitle.Render("sstop")
	timestamp := styleDetailLabel.Render(snap.Timestamp.Format("15:04:05"))

	// Pause indicator
	pauseTag := ""
	if paused {
		pauseTag = stylePaused.Render(" PAUSED ")
	}

	procCount := styleHeaderValue.Render(fmt.Sprintf("%d processes", len(snap.Processes)))

	// Interface indicator
	ifaceTag := ""
	if activeIface != "" {
		ifaceTag = styleFooterKey.Render("["+activeIface+"]") + " "
	} else {
		ifaceTag = styleDetailLabel.Render("[all]") + " "
	}

	// Calculate total up/down based on active interface
	totalUp, totalDown := snap.TotalUp, snap.TotalDown
	if activeIface != "" {
		totalUp, totalDown = 0, 0
		for _, iface := range snap.Interfaces {
			if iface.Name == activeIface {
				totalUp = iface.SendRate
				totalDown = iface.RecvRate
				break
			}
		}
	}

	// Single trend arrow for total bandwidth (up+down combined)
	trendArrow := TrendArrow(snap.TotalRateHistory)
	trendStyled := ""
	switch trendArrow {
	case "↑":
		trendStyled = styleHeaderUp.Render(" ↑")
	case "↓":
		trendStyled = styleHeaderDown.Render(" ↓")
	case "→":
		trendStyled = styleDetailLabel.Render(" →")
	}

	upLabel := styleHeaderUp.Render("▲ " + FormatRate(totalUp))
	downLabel := styleHeaderDown.Render("▼ "+FormatRate(totalDown)) + trendStyled

	left := lipgloss.JoinHorizontal(lipgloss.Center,
		title, "  ", timestamp, pauseTag, "  ", procCount,
	)
	right := lipgloss.JoinHorizontal(lipgloss.Center,
		ifaceTag, upLabel, "  ", downLabel,
	)

	// Pad the space between left and right
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}

	headerLine := left + strings.Repeat(" ", gap) + right

	// Header sparkline — total bandwidth history
	sparklineLine := ""
	if len(snap.TotalRateHistory) > 0 {
		sparkW := 30
		if sparkW > width-4 {
			sparkW = width - 4
		}
		if sparkW > 0 {
			sparkline := Sparkline(snap.TotalRateHistory, sparkW)
			sparklineLine = "  " + styleSparklineActive.Render(sparkline)
		}
	}

	// Interface stats line — show rates for each interface (skip zero-traffic unless active)
	var ifaceParts []string
	for _, iface := range snap.Interfaces {
		// Skip interfaces with no traffic, unless it's the selected interface
		if iface.SendRate == 0 && iface.RecvRate == 0 && activeIface != iface.Name {
			continue
		}
		// Highlight the active interface
		nameStyle := styleDetailLabel
		if activeIface == iface.Name {
			nameStyle = styleFooterKey
		}
		ifaceParts = append(ifaceParts,
			nameStyle.Render(iface.Name+":")+
				" "+styleHeaderUp.Render(FormatRate(iface.SendRate))+
				styleDetailLabel.Render("↑ ")+
				styleHeaderDown.Render(FormatRate(iface.RecvRate))+
				styleDetailLabel.Render("↓"),
		)
	}
	ifaceLine := ""
	if len(ifaceParts) > 0 {
		ifaceLine = strings.Join(ifaceParts, "  ")
		// Truncate if too wide
		if lipgloss.Width(ifaceLine) > width {
			ifaceLine = ""
			for i, part := range ifaceParts {
				test := ifaceLine
				if i > 0 {
					test += "  "
				}
				test += part
				if lipgloss.Width(test) > width-3 {
					ifaceLine += " .."
					break
				}
				ifaceLine = test
			}
		}
	}

	separator := styleBorder.Render(strings.Repeat("─", width))

	var parts []string
	parts = append(parts, headerLine)
	if sparklineLine != "" {
		parts = append(parts, sparklineLine)
	}
	if ifaceLine != "" {
		parts = append(parts, ifaceLine)
	}
	parts = append(parts, separator)

	return strings.Join(parts, "\n")
}
