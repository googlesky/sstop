package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/googlesky/sstop/internal/model"
)

// remoteHostsView manages the remote hosts aggregation view.
type remoteHostsView struct {
	cursor     int
	offset     int
	viewHeight int
}

func newRemoteHostsView() remoteHostsView {
	return remoteHostsView{}
}

func (v *remoteHostsView) moveUp() {
	if v.cursor > 0 {
		v.cursor--
	}
}

func (v *remoteHostsView) moveDown(maxIdx int) {
	if maxIdx < 0 {
		return
	}
	if v.cursor < maxIdx {
		v.cursor++
	}
}

func (v *remoteHostsView) pageUp() {
	v.cursor -= v.viewHeight / 2
	if v.cursor < 0 {
		v.cursor = 0
	}
}

func (v *remoteHostsView) pageDown(maxIdx int) {
	if maxIdx < 0 {
		return
	}
	v.cursor += v.viewHeight / 2
	if v.cursor > maxIdx {
		v.cursor = maxIdx
	}
}

func (v *remoteHostsView) goHome() {
	v.cursor = 0
}

func (v *remoteHostsView) goEnd(maxIdx int) {
	if maxIdx < 0 {
		v.cursor = 0
		return
	}
	v.cursor = maxIdx
}

// Column widths for remote hosts table
const (
	rhUpW    = 12 // bar(5) + gap(1) + text(6)
	rhDownW  = 12 // bar(5) + gap(1) + text(6)
	rhConnsW = 6
	rhProcsW = 20
)

func (v *remoteHostsView) render(hosts []model.RemoteHostSummary, width, height int) string {
	v.viewHeight = height

	if len(hosts) == 0 {
		return styleDetailLabel.Render("  No remote host connections")
	}

	// Find max rates for bar scaling
	maxUp, maxDown := 0.0, 0.0
	for i := range hosts {
		if hosts[i].UpRate > maxUp {
			maxUp = hosts[i].UpRate
		}
		if hosts[i].DownRate > maxDown {
			maxDown = hosts[i].DownRate
		}
	}

	// Dynamic host width
	// Layout: indent(2) + host + 4 gaps between 5 columns (HOST, UP, DOWN, CONNS, PROCS)
	fixedW := 2 + rhUpW + rhDownW + rhConnsW + rhProcsW + 4
	hostW := width - fixedW
	if hostW < 15 {
		hostW = 15
	}

	// Header
	header := v.renderHeader(hostW)

	// Scroll
	if v.cursor < v.offset {
		v.offset = v.cursor
	}
	visibleRows := height - 2 // -2 for title + column header
	if visibleRows < 1 {
		visibleRows = 1
	}
	if v.cursor >= v.offset+visibleRows {
		v.offset = v.cursor - visibleRows + 1
	}

	if v.cursor >= len(hosts) {
		v.cursor = len(hosts) - 1
	}
	if v.cursor < 0 {
		v.cursor = 0
	}

	var lines []string
	lines = append(lines, header)

	end := v.offset + visibleRows
	if end > len(hosts) {
		end = len(hosts)
	}

	for i := v.offset; i < end; i++ {
		h := &hosts[i]
		selected := i == v.cursor
		isEvenRow := (i-v.offset)%2 == 1

		hostName := h.Host
		if hostName == "" && h.IP != nil {
			hostName = h.IP.String()
		}
		if hostName == "" {
			hostName = "unknown"
		}
		hostName = Truncate(hostName, hostW)
		hostName = fmt.Sprintf("%-*s", hostW, hostName)

		barW := 5
		upBar := BandwidthBar(h.UpRate, maxUp, barW)
		downBar := BandwidthBar(h.DownRate, maxDown, barW)
		upText := FormatRateCompact(h.UpRate)   // always 6 chars
		downText := FormatRateCompact(h.DownRate) // always 6 chars

		conns := fmt.Sprintf("%*d", rhConnsW, h.ConnCount)
		procs := Truncate(strings.Join(h.Processes, ","), rhProcsW)
		procs = fmt.Sprintf("%-*s", rhProcsW, procs)

		var row string
		if selected {
			styledHost := styleTableRowSelected.Foreground(colorFg).Bold(true).Render(hostName)
			styledUp := styleTableRowSelected.Foreground(colorGreen).Render(upBar + " " + upText)
			styledDown := styleTableRowSelected.Foreground(colorRed).Render(downBar + " " + downText)
			styledConns := styleTableRowSelected.Foreground(colorCyan).Render(conns)
			styledProcs := styleTableRowSelected.Foreground(colorFgDim).Render(procs)
			row = lipgloss.JoinHorizontal(lipgloss.Top,
				styleTableRowSelected.Render("▸ "),
				styledHost, " ",
				styledUp, " ", styledDown, " ",
				styledConns, " ", styledProcs,
			)
			rowWidth := lipgloss.Width(row)
			if rowWidth < width {
				row += styleTableRowSelected.Render(strings.Repeat(" ", width-rowWidth))
			}
		} else {
			bgStyle := lipgloss.NewStyle()
			hostStyle := styleProcessName
			upTextStyle := styleUpRate
			downTextStyle := styleDownRate
			connsStyle := styleConnCount
			procsStyle := styleDetailLabel
			upBarStyled := barStyleUp(h.UpRate, maxUp).Render(upBar)
			downBarStyled := barStyleDown(h.DownRate, maxDown).Render(downBar)

			if isEvenRow {
				bgStyle = styleZebraRow
				hostStyle = hostStyle.Background(colorZebraRow)
				upTextStyle = upTextStyle.Background(colorZebraRow)
				downTextStyle = downTextStyle.Background(colorZebraRow)
				connsStyle = connsStyle.Background(colorZebraRow)
				procsStyle = procsStyle.Background(colorZebraRow)
				upBarStyled = barStyleUp(h.UpRate, maxUp).Background(colorZebraRow).Render(upBar)
				downBarStyled = barStyleDown(h.DownRate, maxDown).Background(colorZebraRow).Render(downBar)
			}

			row = lipgloss.JoinHorizontal(lipgloss.Top,
				bgStyle.Render("  "),
				hostStyle.Render(hostName), bgStyle.Render(" "),
				upBarStyled, bgStyle.Render(" "), upTextStyle.Render(upText), bgStyle.Render(" "),
				downBarStyled, bgStyle.Render(" "), downTextStyle.Render(downText), bgStyle.Render(" "),
				connsStyle.Render(conns), bgStyle.Render(" "),
				procsStyle.Render(procs),
			)

			if isEvenRow {
				rowWidth := lipgloss.Width(row)
				if rowWidth < width {
					row += bgStyle.Render(strings.Repeat(" ", width-rowWidth))
				}
			}
		}

		lines = append(lines, row)
	}

	return strings.Join(lines, "\n")
}

func (v *remoteHostsView) renderHeader(hostW int) string {
	title := styleTitle.Render("  Remote Hosts")
	cols := lipgloss.JoinHorizontal(lipgloss.Top,
		"  ",
		styleTableHeader.Render(fmt.Sprintf("%-*s", hostW, "HOST")), " ",
		styleTableHeader.Render(fmt.Sprintf("%*s", rhUpW, "UPLOAD/s")), " ",
		styleTableHeader.Render(fmt.Sprintf("%*s", rhDownW, "DOWNLOAD/s")), " ",
		styleTableHeader.Render(fmt.Sprintf("%*s", rhConnsW, "CONNS")), " ",
		styleTableHeader.Render(fmt.Sprintf("%-*s", rhProcsW, "PROCESSES")),
	)
	return title + "\n" + cols
}
