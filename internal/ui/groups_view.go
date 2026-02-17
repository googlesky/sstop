package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/googlesky/sstop/internal/model"
)

// groupEntry represents an aggregated process group (container/service/user).
type groupEntry struct {
	Name      string  // display name
	Type      string  // "docker", "podman", "systemd", "user"
	ProcCount int     // number of processes in this group
	UpRate    float64 // aggregate upload rate
	DownRate  float64 // aggregate download rate
	ConnCount int     // total connections
}

// groupsView manages the container/service group view.
type groupsView struct {
	cursor     int
	offset     int
	viewHeight int
}

func (v *groupsView) moveUp() {
	if v.cursor > 0 {
		v.cursor--
	}
}

func (v *groupsView) moveDown(maxIdx int) {
	if maxIdx < 0 {
		return
	}
	if v.cursor < maxIdx {
		v.cursor++
	}
}

func (v *groupsView) pageUp() {
	v.cursor -= v.viewHeight / 2
	if v.cursor < 0 {
		v.cursor = 0
	}
}

func (v *groupsView) pageDown(maxIdx int) {
	if maxIdx < 0 {
		return
	}
	v.cursor += v.viewHeight / 2
	if v.cursor > maxIdx {
		v.cursor = maxIdx
	}
}

func (v *groupsView) goHome() {
	v.cursor = 0
}

func (v *groupsView) goEnd(maxIdx int) {
	if maxIdx < 0 {
		return
	}
	v.cursor = maxIdx
}

// classifyGroup determines the group name and type for a process.
func classifyGroup(proc *model.ProcessSummary) (name, typ string) {
	if proc.ContainerID != "" {
		// Docker or Podman — we can't easily distinguish without more info,
		// so just call it "container"
		return proc.ContainerID, "container"
	}
	if proc.ServiceName != "" {
		return proc.ServiceName, "systemd"
	}
	return "other", "user"
}

// buildGroups aggregates processes into groups.
func buildGroups(procs []model.ProcessSummary) []groupEntry {
	type agg struct {
		name      string
		typ       string
		procCount int
		upRate    float64
		downRate  float64
		connCount int
	}
	groups := make(map[string]*agg)

	for i := range procs {
		name, typ := classifyGroup(&procs[i])
		key := typ + ":" + name
		g, ok := groups[key]
		if !ok {
			g = &agg{name: name, typ: typ}
			groups[key] = g
		}
		g.procCount++
		g.upRate += procs[i].UpRate
		g.downRate += procs[i].DownRate
		g.connCount += procs[i].ConnCount
	}

	result := make([]groupEntry, 0, len(groups))
	for _, g := range groups {
		result = append(result, groupEntry{
			Name:      g.name,
			Type:      g.typ,
			ProcCount: g.procCount,
			UpRate:    g.upRate,
			DownRate:  g.downRate,
			ConnCount: g.connCount,
		})
	}

	// Sort by total rate descending
	sort.Slice(result, func(i, j int) bool {
		ti := result[i].UpRate + result[i].DownRate
		tj := result[j].UpRate + result[j].DownRate
		return ti > tj
	})

	return result
}

func (v *groupsView) render(procs []model.ProcessSummary, width, height int) string {
	groups := buildGroups(procs)

	v.viewHeight = height

	// Clamp cursor if groups count changed
	if len(groups) > 0 && v.cursor >= len(groups) {
		v.cursor = len(groups) - 1
	}
	if v.cursor < 0 {
		v.cursor = 0
	}

	// Title
	title := styleTitle.Render("  Groups (Docker / Systemd)")
	titleLine := title

	// Column widths
	// GROUP | TYPE | PROCS | UPLOAD/s | DOWNLOAD/s | CONNS
	typeW := 10
	procsW := 6
	upW := 8
	downW := 8
	connsW := 6
	fixedW := typeW + procsW + upW + downW + connsW + 7 // 7 for separators/padding
	nameW := width - fixedW
	if nameW < 10 {
		nameW = 10
	}

	// Header
	headerLine := fmt.Sprintf("  %-*s %-*s %*s %*s %*s %*s",
		nameW, "GROUP",
		typeW, "TYPE",
		procsW, "PROCS",
		upW, "UP/s",
		downW, "DOWN/s",
		connsW, "CONNS",
	)
	headerStyled := styleTableHeader.Render(headerLine)

	// Available rows
	rowsAvail := height - 2 // title + header
	if rowsAvail < 1 {
		rowsAvail = 1
	}

	// Adjust offset
	if v.cursor < v.offset {
		v.offset = v.cursor
	}
	if v.cursor >= v.offset+rowsAvail {
		v.offset = v.cursor - rowsAvail + 1
	}

	if len(groups) == 0 {
		empty := styleDetailLabel.Render("  No active processes")
		return strings.Join([]string{titleLine, headerStyled, empty}, "\n")
	}

	var rows []string
	end := v.offset + rowsAvail
	if end > len(groups) {
		end = len(groups)
	}

	for idx := v.offset; idx < end; idx++ {
		g := groups[idx]

		name := truncateStr(g.Name, nameW)
		typStr := truncateStr(g.Type, typeW)
		upStr := FormatRateCompact(g.UpRate)
		downStr := FormatRateCompact(g.DownRate)

		line := fmt.Sprintf("  %-*s %-*s %*d %*s %*s %*d",
			nameW, name,
			typeW, typStr,
			procsW, g.ProcCount,
			upW, upStr,
			downW, downStr,
			connsW, g.ConnCount,
		)

		var rowStyle lipgloss.Style
		if idx == v.cursor {
			rowStyle = styleTableRowSelected
		} else if idx%2 == 1 {
			rowStyle = styleZebraRow
		} else {
			rowStyle = styleTableRow
		}

		// Apply rate coloring
		styledLine := rowStyle.Render(line)
		rows = append(rows, styledLine)
	}

	var parts []string
	parts = append(parts, titleLine)
	parts = append(parts, headerStyled)
	parts = append(parts, rows...)

	return strings.Join(parts, "\n")
}

// truncateStr truncates s to maxLen, adding ".." if truncated.
func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 2 {
		return s[:maxLen]
	}
	return s[:maxLen-2] + ".."
}
