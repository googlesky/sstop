package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/googlesky/sstop/internal/model"
)

// listenPortsView manages the system-wide listening ports view.
type listenPortsView struct {
	cursor     int
	offset     int
	viewHeight int
}

func newListenPortsView() listenPortsView {
	return listenPortsView{}
}

func (v *listenPortsView) moveUp() {
	if v.cursor > 0 {
		v.cursor--
	}
}

func (v *listenPortsView) moveDown(maxIdx int) {
	if maxIdx < 0 {
		return
	}
	if v.cursor < maxIdx {
		v.cursor++
	}
}

func (v *listenPortsView) pageUp() {
	v.cursor -= v.viewHeight / 2
	if v.cursor < 0 {
		v.cursor = 0
	}
}

func (v *listenPortsView) pageDown(maxIdx int) {
	if maxIdx < 0 {
		return
	}
	v.cursor += v.viewHeight / 2
	if v.cursor > maxIdx {
		v.cursor = maxIdx
	}
}

func (v *listenPortsView) goHome() {
	v.cursor = 0
}

func (v *listenPortsView) goEnd(maxIdx int) {
	if maxIdx < 0 {
		v.cursor = 0
		return
	}
	v.cursor = maxIdx
}

// Column widths
const (
	lpProtoW = 5
	lpPidW   = 8
	lpProcW  = 20
)

func (v *listenPortsView) render(ports []model.ListenPortEntry, width, height int) string {
	v.viewHeight = height

	if len(ports) == 0 {
		return styleDetailLabel.Render("  No listening ports")
	}

	// Dynamic address width
	// 4 columns (PROTO, ADDR, PID, PROCESS) = 3 gaps + 2 indent
	fixedW := lpProtoW + lpPidW + lpProcW + 3 + 2
	addrW := width - fixedW
	cmdW := 0
	if addrW > 40 {
		// Split remaining: address gets half, cmdline gets half
		cmdW = addrW / 3
		addrW = addrW - cmdW - 1 // -1 for gap
	}
	if addrW < 15 {
		addrW = 15
	}

	// Title + header
	title := styleTitle.Render(fmt.Sprintf("  Listening Ports (%d)", len(ports)))
	header := v.renderHeader(addrW, cmdW)

	// Scroll
	if v.cursor >= len(ports) {
		v.cursor = len(ports) - 1
	}
	if v.cursor < 0 {
		v.cursor = 0
	}
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

	var lines []string
	lines = append(lines, title)
	lines = append(lines, header)

	end := v.offset + visibleRows
	if end > len(ports) {
		end = len(ports)
	}

	for i := v.offset; i < end; i++ {
		lp := &ports[i]
		selected := i == v.cursor
		isEvenRow := (i-v.offset)%2 == 1

		proto := lp.Proto.String()

		// Format local address
		addr := "*"
		if lp.IP != nil && !lp.IP.IsUnspecified() {
			addr = lp.IP.String()
		}
		addr = fmt.Sprintf("%s:%d", addr, lp.Port)
		addr = Truncate(addr, addrW)
		addr = fmt.Sprintf("%-*s", addrW, addr)

		pid := fmt.Sprintf("%-*d", lpPidW, lp.PID)
		proc := Truncate(lp.Process, lpProcW)
		proc = fmt.Sprintf("%-*s", lpProcW, proc)

		cmdline := ""
		if cmdW > 0 {
			cmdline = Truncate(lp.Cmdline, cmdW)
			cmdline = fmt.Sprintf("%-*s", cmdW, cmdline)
		}

		var row string
		if selected {
			styledProto := styleTableRowSelected.Foreground(colorCyan).Render(fmt.Sprintf("%-*s", lpProtoW, proto))
			styledAddr := styleTableRowSelected.Foreground(colorFg).Render(addr)
			styledPid := styleTableRowSelected.Foreground(colorFgDim).Render(pid)
			styledProc := styleTableRowSelected.Foreground(colorFg).Bold(true).Render(proc)
			row = lipgloss.JoinHorizontal(lipgloss.Top,
				styleTableRowSelected.Render("▸ "),
				styledProto, " ",
				styledAddr, " ",
				styledPid, " ",
				styledProc,
			)
			if cmdW > 0 {
				row += " " + styleTableRowSelected.Foreground(colorFgDim).Render(cmdline)
			}
			rowWidth := lipgloss.Width(row)
			if rowWidth < width {
				row += styleTableRowSelected.Render(strings.Repeat(" ", width-rowWidth))
			}
		} else {
			bgStyle := lipgloss.NewStyle()
			protoStyle := styleStateListen
			addrStyle := styleHeaderValue
			pidStyle := stylePID
			procStyle := styleProcessName
			cmdStyle := styleDetailLabel

			if isEvenRow {
				bgStyle = styleZebraRow
				protoStyle = protoStyle.Background(colorZebraRow)
				addrStyle = addrStyle.Background(colorZebraRow)
				pidStyle = pidStyle.Background(colorZebraRow)
				procStyle = procStyle.Background(colorZebraRow)
				cmdStyle = cmdStyle.Background(colorZebraRow)
			}

			row = lipgloss.JoinHorizontal(lipgloss.Top,
				bgStyle.Render("  "),
				protoStyle.Render(fmt.Sprintf("%-*s", lpProtoW, proto)), bgStyle.Render(" "),
				addrStyle.Render(addr), bgStyle.Render(" "),
				pidStyle.Render(pid), bgStyle.Render(" "),
				procStyle.Render(proc),
			)
			if cmdW > 0 {
				row += bgStyle.Render(" ") + cmdStyle.Render(cmdline)
			}

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

func (v *listenPortsView) renderHeader(addrW, cmdW int) string {
	parts := []string{
		"  ",
		styleTableHeader.Render(fmt.Sprintf("%-*s", lpProtoW, "PROTO")), " ",
		styleTableHeader.Render(fmt.Sprintf("%-*s", addrW, "LOCAL ADDRESS")), " ",
		styleTableHeader.Render(fmt.Sprintf("%-*s", lpPidW, "PID")), " ",
		styleTableHeader.Render(fmt.Sprintf("%-*s", lpProcW, "PROCESS")),
	}
	if cmdW > 0 {
		parts = append(parts, " ", styleTableHeader.Render(fmt.Sprintf("%-*s", cmdW, "COMMAND")))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}
