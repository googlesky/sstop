package ui

import (
	"fmt"
	"net"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/googlesky/sstop/internal/model"
)

// processDetail manages the detail view for a single process.
type processDetail struct {
	pid        uint32
	cursor     int
	offset     int
	viewHeight int
	showDNS    bool // toggle between hostname and raw IP
}

func newProcessDetail(pid uint32) processDetail {
	return processDetail{pid: pid, showDNS: true}
}

func (d *processDetail) moveUp() {
	if d.cursor > 0 {
		d.cursor--
	}
}

func (d *processDetail) moveDown(maxIdx int) {
	if maxIdx < 0 {
		return
	}
	if d.cursor < maxIdx {
		d.cursor++
	}
}

func (d *processDetail) pageUp() {
	d.cursor -= d.viewHeight / 2
	if d.cursor < 0 {
		d.cursor = 0
	}
}

func (d *processDetail) pageDown(maxIdx int) {
	if maxIdx < 0 {
		return
	}
	d.cursor += d.viewHeight / 2
	if d.cursor > maxIdx {
		d.cursor = maxIdx
	}
}

func (d *processDetail) toggleDNS() {
	d.showDNS = !d.showDNS
}

// connColumnLayout computes dynamic column widths based on terminal width.
type connColumnLayout struct {
	protoW  int
	localW  int
	remoteW int
	stateW  int
	ageW    int
	upW     int
	downW   int
}

func computeConnLayout(width int) connColumnLayout {
	const (
		protoW = 5
		stateW = 10 // shortened to fit badges
		ageW   = 7
		upW    = 10
		downW  = 10
		fixed  = protoW + stateW + ageW + upW + downW + 6 + 2 // 6 gaps between 7 columns + 2 indent
	)

	remaining := width - fixed
	if remaining < 30 {
		remaining = 30
	}

	// REMOTE gets 60%, LOCAL gets 40% (remote hosts are typically longer)
	remoteW := remaining * 60 / 100
	localW := remaining - remoteW

	return connColumnLayout{
		protoW:  protoW,
		localW:  localW,
		remoteW: remoteW,
		stateW:  stateW,
		ageW:    ageW,
		upW:     upW,
		downW:   downW,
	}
}

// stateBadge returns a compact badge with icon for a TCP state.
func stateBadge(s model.SocketState) string {
	switch s {
	case model.StateEstablished:
		return "⚡ESTAB"
	case model.StateTimeWait:
		return "⏳T_WAIT"
	case model.StateCloseWait:
		return "⚠ C_WAIT"
	case model.StateSynSent:
		return "◌ SYN_S"
	case model.StateSynRecv:
		return "◌ SYN_R"
	case model.StateListen:
		return "● LISTEN"
	case model.StateFinWait1:
		return "◌ FIN_1"
	case model.StateFinWait2:
		return "◌ FIN_2"
	case model.StateClosing:
		return "◌ CLOSNG"
	case model.StateLastAck:
		return "◌ LSTACK"
	case model.StateClose:
		return "◌ CLOSE"
	default:
		return "? " + s.String()
	}
}

func (d *processDetail) render(proc *model.ProcessSummary, width, height int) string {
	if proc == nil {
		return styleDetailLabel.Render("  Process not found")
	}

	d.viewHeight = height
	lay := computeConnLayout(width)

	var lines []string

	// Process info header
	infoLine := lipgloss.JoinHorizontal(lipgloss.Center,
		styleTitle.Render(fmt.Sprintf(" %s", proc.Name)),
		styleDetailLabel.Render(fmt.Sprintf("  PID: %d", proc.PID)),
		"  ",
		styleHeaderUp.Render("▲ "+FormatRate(proc.UpRate)),
		"  ",
		styleHeaderDown.Render("▼ "+FormatRate(proc.DownRate)),
	)
	lines = append(lines, infoLine)

	// Cmdline
	if proc.Cmdline != "" {
		cmdline := Truncate(proc.Cmdline, width-4)
		lines = append(lines, styleDetailLabel.Render("  "+cmdline))
	}

	lines = append(lines, styleBorder.Render(strings.Repeat("─", width)))

	// Listening ports
	if len(proc.ListenPorts) > 0 {
		lines = append(lines, styleTitle.Render("  Listening Ports"))
		for _, lp := range proc.ListenPorts {
			addr := "*"
			if lp.IP != nil && !lp.IP.IsUnspecified() {
				addr = lp.IP.String()
			}
			lines = append(lines,
				"  "+styleStateListen.Render(fmt.Sprintf("  ● %s %s:%d", lp.Proto, addr, lp.Port)),
			)
		}
		lines = append(lines, "")
	}

	// Connections table
	if len(proc.Connections) > 0 {
		lines = append(lines, styleTitle.Render(
			fmt.Sprintf("  Connections (%d)", len(proc.Connections)),
		))

		// Connection table header with dynamic widths
		connHeader := fmt.Sprintf("  %-*s %-*s %-*s %-*s %*s %*s %*s",
			lay.protoW, "PROTO",
			lay.localW, "LOCAL",
			lay.remoteW, "REMOTE",
			lay.stateW, "STATE",
			lay.ageW, "AGE",
			lay.upW, "UP/s",
			lay.downW, "DOWN/s")
		lines = append(lines, styleTableHeader.Render(connHeader))

		// Calculate scroll
		headerLines := len(lines)
		availRows := height - headerLines - 1
		if availRows < 1 {
			availRows = 1
		}

		maxIdx := len(proc.Connections) - 1
		if d.cursor > maxIdx {
			d.cursor = maxIdx
		}
		if d.cursor < 0 {
			d.cursor = 0
		}

		if d.cursor < d.offset {
			d.offset = d.cursor
		}
		if d.cursor >= d.offset+availRows {
			d.offset = d.cursor - availRows + 1
		}

		end := d.offset + availRows
		if end > len(proc.Connections) {
			end = len(proc.Connections)
		}

		for i := d.offset; i < end; i++ {
			c := &proc.Connections[i]
			selected := i == d.cursor

			proto := c.Proto.String()
			local := formatConnAddr(c.SrcIP, c.SrcPort)
			remote := d.formatRemote(c)
			state := stateBadge(c.State)
			age := FormatAge(c.Age)
			up := FormatRate(c.UpRate)
			down := FormatRate(c.DownRate)

			local = Truncate(local, lay.localW)
			remote = Truncate(remote, lay.remoteW)

			stateStyle := stateToStyle(c.State)

			indicator := "  "
			rowStyle := styleTableRow
			if selected {
				indicator = "▸ "
				rowStyle = styleTableRowSelected
			}

			row := lipgloss.JoinHorizontal(lipgloss.Top,
				rowStyle.Render(indicator),
				rowStyle.Render(fmt.Sprintf("%-*s ", lay.protoW, proto)),
				rowStyle.Render(fmt.Sprintf("%-*s ", lay.localW, local)),
				rowStyle.Render(fmt.Sprintf("%-*s ", lay.remoteW, remote)),
				stateStyle.Render(fmt.Sprintf("%-*s ", lay.stateW, state)),
				styleDetailLabel.Render(fmt.Sprintf("%*s ", lay.ageW, age)),
				styleUpRate.Render(fmt.Sprintf("%*s ", lay.upW, up)),
				styleDownRate.Render(fmt.Sprintf("%*s", lay.downW, down)),
			)

			if selected {
				rowWidth := lipgloss.Width(row)
				if rowWidth < width {
					row += rowStyle.Render(strings.Repeat(" ", width-rowWidth))
				}
			}

			lines = append(lines, row)
		}
	} else if len(proc.ListenPorts) == 0 {
		lines = append(lines, styleDetailLabel.Render("  No active connections"))
	}

	return strings.Join(lines, "\n")
}

// formatRemote formats the remote address, preferring hostname when showDNS is on.
func (d *processDetail) formatRemote(c *model.Connection) string {
	if d.showDNS && c.RemoteHost != "" {
		return fmt.Sprintf("%s:%d", c.RemoteHost, c.DstPort)
	}
	return formatConnAddr(c.DstIP, c.DstPort)
}

func formatConnAddr(ip net.IP, port uint16) string {
	if ip == nil || ip.IsUnspecified() {
		return fmt.Sprintf("*:%d", port)
	}
	if ip4 := ip.To4(); ip4 != nil {
		return fmt.Sprintf("%s:%d", ip4, port)
	}
	return fmt.Sprintf("[%s]:%d", ip, port)
}

func stateToStyle(s model.SocketState) lipgloss.Style {
	switch s {
	case model.StateEstablished:
		return styleStateEstablished
	case model.StateListen:
		return styleStateListen
	case model.StateTimeWait:
		return styleStateTimeWait
	case model.StateFinWait1, model.StateFinWait2, model.StateClosing, model.StateCloseWait, model.StateLastAck:
		return styleStateClosing
	default:
		return styleStateOther
	}
}
