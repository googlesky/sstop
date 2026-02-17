package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/googlesky/sstop/internal/model"
)

// SortColumn defines which column to sort by.
type SortColumn int

const (
	SortByRate  SortColumn = iota // total bandwidth (default)
	SortByDown                    // download rate
	SortByUp                      // upload rate
	SortByPID                     // PID
	SortByName                    // process name
	SortByConns                   // connection count
	sortColumnCount
)

var sortColumnNames = [...]string{
	"RATE", "DOWN", "UP", "PID", "NAME", "CONNS",
}

func (s SortColumn) String() string {
	if int(s) < len(sortColumnNames) {
		return sortColumnNames[s]
	}
	return "?"
}

// processTable manages the process list view state.
type processTable struct {
	cursor         int
	offset         int // scroll offset
	sortCol        SortColumn
	filter         string
	processes      []model.ProcessSummary
	filtered       []model.ProcessSummary
	viewHeight     int
	cumulativeMode bool
	treeMode       bool
	treePrefix     map[uint32]string // PID → tree drawing prefix
}

func newProcessTable() processTable {
	return processTable{
		sortCol: SortByRate,
	}
}

func (t *processTable) update(processes []model.ProcessSummary) {
	t.processes = processes
	t.applyFilterAndSort()

	// Keep cursor in bounds
	if t.cursor >= len(t.filtered) {
		t.cursor = len(t.filtered) - 1
	}
	if t.cursor < 0 {
		t.cursor = 0
	}
}

func (t *processTable) applyFilterAndSort() {
	// Filter
	if t.filter == "" {
		t.filtered = make([]model.ProcessSummary, len(t.processes))
		copy(t.filtered, t.processes)
	} else {
		t.filtered = t.filtered[:0]
		f := ParseFilter(t.filter)
		for i := range t.processes {
			if f.Match(&t.processes[i]) {
				t.filtered = append(t.filtered, t.processes[i])
			}
		}
	}

	// Sort
	sort.SliceStable(t.filtered, func(i, j int) bool {
		a, b := &t.filtered[i], &t.filtered[j]
		if t.cumulativeMode {
			switch t.sortCol {
			case SortByRate:
				return (a.CumUp + a.CumDown) > (b.CumUp + b.CumDown)
			case SortByDown:
				return a.CumDown > b.CumDown
			case SortByUp:
				return a.CumUp > b.CumUp
			case SortByPID:
				return a.PID < b.PID
			case SortByName:
				return strings.ToLower(a.Name) < strings.ToLower(b.Name)
			case SortByConns:
				return a.ConnCount > b.ConnCount
			default:
				return false
			}
		}
		switch t.sortCol {
		case SortByRate:
			return (a.UpRate + a.DownRate) > (b.UpRate + b.DownRate)
		case SortByDown:
			return a.DownRate > b.DownRate
		case SortByUp:
			return a.UpRate > b.UpRate
		case SortByPID:
			return a.PID < b.PID
		case SortByName:
			return strings.ToLower(a.Name) < strings.ToLower(b.Name)
		case SortByConns:
			return a.ConnCount > b.ConnCount
		default:
			return false
		}
	})

	// Apply tree ordering if tree mode is active
	t.buildTree()
}

// treeNode represents a process in the tree with its indentation info.
type treeNode struct {
	proc   model.ProcessSummary
	depth  int
	prefix string // tree drawing prefix: "├── ", "└── ", "│   ", "    "
}

// buildTree reorders filtered processes into tree order.
func (t *processTable) buildTree() {
	if !t.treeMode || len(t.filtered) == 0 {
		return
	}

	// Build PID set of processes in filtered list
	pidSet := make(map[uint32]bool)
	byPID := make(map[uint32]*model.ProcessSummary)
	for i := range t.filtered {
		pidSet[t.filtered[i].PID] = true
		p := t.filtered[i]
		byPID[p.PID] = &p
	}

	// Build children map
	children := make(map[uint32][]uint32) // PPID → [child PIDs]
	var roots []uint32
	for _, p := range t.filtered {
		ppid := p.PPID
		if ppid == 0 || !pidSet[ppid] {
			roots = append(roots, p.PID)
		} else {
			children[ppid] = append(children[ppid], p.PID)
		}
	}

	// Sort roots by current sort order (they're already sorted)
	// DFS to build tree-ordered list
	result := make([]model.ProcessSummary, 0, len(t.filtered))
	treeInfo := make(map[uint32]string) // PID → prefix string

	var walk func(pid uint32, depth int, prefix string, isLast bool)
	walk = func(pid uint32, depth int, prefix string, isLast bool) {
		p := byPID[pid]
		if p == nil {
			return
		}

		// Set tree prefix for this node
		nodePrefix := ""
		if depth > 0 {
			if isLast {
				nodePrefix = prefix + "└─"
			} else {
				nodePrefix = prefix + "├─"
			}
		}
		treeInfo[pid] = nodePrefix
		result = append(result, *p)

		// Walk children
		kids := children[pid]
		childPrefix := prefix
		if depth > 0 {
			if isLast {
				childPrefix = prefix + "  "
			} else {
				childPrefix = prefix + "│ "
			}
		}
		for i, kid := range kids {
			walk(kid, depth+1, childPrefix, i == len(kids)-1)
		}
	}

	for _, rootPID := range roots {
		walk(rootPID, 0, "", true)
	}

	t.filtered = result
	t.treePrefix = treeInfo
}

func (t *processTable) nextSort() {
	t.sortCol = (t.sortCol + 1) % sortColumnCount
	t.applyFilterAndSort()
}

func (t *processTable) moveUp() {
	if t.cursor > 0 {
		t.cursor--
	}
}

func (t *processTable) moveDown() {
	if t.cursor < len(t.filtered)-1 {
		t.cursor++
	}
}

func (t *processTable) pageUp() {
	t.cursor -= t.viewHeight / 2
	if t.cursor < 0 {
		t.cursor = 0
	}
}

func (t *processTable) pageDown() {
	t.cursor += t.viewHeight / 2
	if t.cursor >= len(t.filtered) {
		t.cursor = len(t.filtered) - 1
	}
	if t.cursor < 0 {
		t.cursor = 0
	}
}

func (t *processTable) goHome() {
	t.cursor = 0
}

func (t *processTable) goEnd() {
	t.cursor = len(t.filtered) - 1
	if t.cursor < 0 {
		t.cursor = 0
	}
}

func (t *processTable) selected() *model.ProcessSummary {
	if t.cursor >= 0 && t.cursor < len(t.filtered) {
		return &t.filtered[t.cursor]
	}
	return nil
}

// Column widths
const (
	colPidW    = 8
	colUpW     = 12 // bar(5) + gap(1) + text(6)
	colDownW   = 12 // bar(5) + gap(1) + text(6)
	colConnsW  = 6
	colListenW = 6
	colGraphW  = 16 // sparkline width
)

func (t *processTable) render(width, height int, cumulativeMode bool) string {
	t.viewHeight = height

	if len(t.filtered) == 0 {
		return styleDetailLabel.Render("  No processes with network activity")
	}

	// Find max rates for bar scaling
	maxUp, maxDown := 0.0, 0.0
	for i := range t.filtered {
		if cumulativeMode {
			if float64(t.filtered[i].CumUp) > maxUp {
				maxUp = float64(t.filtered[i].CumUp)
			}
			if float64(t.filtered[i].CumDown) > maxDown {
				maxDown = float64(t.filtered[i].CumDown)
			}
		} else {
			if t.filtered[i].UpRate > maxUp {
				maxUp = t.filtered[i].UpRate
			}
			if t.filtered[i].DownRate > maxDown {
				maxDown = t.filtered[i].DownRate
			}
		}
	}

	// Dynamic name width: fill remaining space
	// 6 gaps between 7 header columns + 2 indent
	fixedW := colPidW + colGraphW + colUpW + colDownW + colConnsW + colListenW + 6 + 2
	nameW := width - fixedW
	if nameW < 10 {
		nameW = 10
	}

	// Header
	header := renderTableHeader(nameW, t.sortCol, cumulativeMode)

	// Adjust scroll offset
	if t.cursor < t.offset {
		t.offset = t.cursor
	}
	visibleRows := height - 1 // -1 for header
	if visibleRows < 1 {
		visibleRows = 1
	}
	if t.cursor >= t.offset+visibleRows {
		t.offset = t.cursor - visibleRows + 1
	}

	var lines []string
	lines = append(lines, header)

	end := t.offset + visibleRows
	if end > len(t.filtered) {
		end = len(t.filtered)
	}

	for i := t.offset; i < end; i++ {
		p := &t.filtered[i]
		selected := i == t.cursor
		isEvenRow := (i-t.offset)%2 == 1 // alternate rows for zebra striping

		pid := fmt.Sprintf("%-*d", colPidW, p.PID)
		displayName := p.Name
		if t.treeMode {
			if prefix, ok := t.treePrefix[p.PID]; ok && prefix != "" {
				displayName = prefix + displayName
			}
		}
		name := Truncate(displayName, nameW)
		name = fmt.Sprintf("%-*s", nameW, name)
		graph := Sparkline(p.RateHistory, colGraphW)

		// Bandwidth bars integrated with rate/cumulative text
		barW := 5 // width for the bar portion
		var upVal, downVal float64
		var upText, downText string
		if cumulativeMode {
			upVal = float64(p.CumUp)
			downVal = float64(p.CumDown)
			upText = FormatBytesCompact(p.CumUp)
			downText = FormatBytesCompact(p.CumDown)
		} else {
			upVal = p.UpRate
			downVal = p.DownRate
			upText = FormatRateCompact(p.UpRate)
			downText = FormatRateCompact(p.DownRate)
		}
		upBar := BandwidthBar(upVal, maxUp, barW)
		downBar := BandwidthBar(downVal, maxDown, barW)

		conns := fmt.Sprintf("%*d", colConnsW, p.ConnCount)
		listen := fmt.Sprintf("%*d", colListenW, p.ListenCount)

		var row string
		if selected {
			styledPid := styleTableRowSelected.Foreground(colorFgDim).Render(pid)
			styledName := styleTableRowSelected.Foreground(colorFg).Bold(true).Render(name)
			styledGraph := styleTableRowSelected.Foreground(colorCyan).Render(graph)
			styledUp := styleTableRowSelected.Foreground(colorGreen).Render(upBar + " " + upText)
			styledDown := styleTableRowSelected.Foreground(colorRed).Render(downBar + " " + downText)
			styledConns := styleTableRowSelected.Foreground(colorCyan).Render(conns)
			styledListen := styleTableRowSelected.Foreground(colorMagenta).Render(listen)
			row = lipgloss.JoinHorizontal(lipgloss.Top,
				styleTableRowSelected.Render("▸ "),
				styledPid, " ", styledName, " ",
				styledGraph, " ",
				styledUp, " ", styledDown, " ",
				styledConns, " ", styledListen,
			)
			// Pad to full width with selection background
			rowWidth := lipgloss.Width(row)
			if rowWidth < width {
				row += styleTableRowSelected.Render(strings.Repeat(" ", width-rowWidth))
			}
		} else {
			// Color the sparkline based on activity
			graphStyle := styleSparkline
			if p.UpRate+p.DownRate > 0 {
				graphStyle = styleSparklineActive
			}

			// Rate-intensity colored bars
			upBarStyled := barStyleUp(upVal, maxUp).Render(upBar)
			downBarStyled := barStyleDown(downVal, maxDown).Render(downBar)

			// Zebra striping
			bgStyle := lipgloss.NewStyle()
			pidStyle := stylePID
			nameStyle := styleProcessName
			upTextStyle := styleUpRate
			downTextStyle := styleDownRate
			connsStyle := styleConnCount
			listenStyle := styleListenCount
			if isEvenRow {
				bgStyle = styleZebraRow
				pidStyle = pidStyle.Background(colorZebraRow)
				nameStyle = nameStyle.Background(colorZebraRow)
				graphStyle = graphStyle.Background(colorZebraRow)
				upTextStyle = upTextStyle.Background(colorZebraRow)
				downTextStyle = downTextStyle.Background(colorZebraRow)
				connsStyle = connsStyle.Background(colorZebraRow)
				listenStyle = listenStyle.Background(colorZebraRow)
				upBarStyled = barStyleUp(upVal, maxUp).Background(colorZebraRow).Render(upBar)
				downBarStyled = barStyleDown(downVal, maxDown).Background(colorZebraRow).Render(downBar)
			}

			row = lipgloss.JoinHorizontal(lipgloss.Top,
				bgStyle.Render("  "),
				pidStyle.Render(pid), bgStyle.Render(" "),
				nameStyle.Render(name), bgStyle.Render(" "),
				graphStyle.Render(graph), bgStyle.Render(" "),
				upBarStyled, bgStyle.Render(" "), upTextStyle.Render(upText), bgStyle.Render(" "),
				downBarStyled, bgStyle.Render(" "), downTextStyle.Render(downText), bgStyle.Render(" "),
				connsStyle.Render(conns), bgStyle.Render(" "),
				listenStyle.Render(listen),
			)

			// Pad zebra rows to full width
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

func renderTableHeader(nameW int, sortCol SortColumn, cumulativeMode bool) string {
	upHeader, downHeader := "UPLOAD/s", "DOWNLOAD/s"
	if cumulativeMode {
		upHeader = "UP TOTAL"
		downHeader = "DN TOTAL"
	}

	cols := []struct {
		name  string
		width int
		col   SortColumn
		align int // 0=left, 1=right
	}{
		{"PID", colPidW, SortByPID, 0},
		{"PROCESS", nameW, SortByName, 0},
		{"GRAPH", colGraphW, SortColumn(-1), 0},
		{upHeader, colUpW, SortByUp, 1},
		{downHeader, colDownW, SortByDown, 1},
		{"CONNS", colConnsW, SortByConns, 1},
		{"LISTEN", colListenW, SortColumn(-1), 1},
	}

	var parts []string
	parts = append(parts, "  ") // indent matching row "▸ "

	for i, c := range cols {
		var s string
		if c.align == 1 {
			// Right-aligned
			label := c.name
			if c.col == sortCol {
				label = label + "▾"
			}
			formatted := fmt.Sprintf("%*s", c.width, label)
			if c.col == sortCol {
				s = styleSortIndicator.Render(formatted)
			} else {
				s = styleTableHeader.Render(formatted)
			}
		} else {
			// Left-aligned
			label := c.name
			if c.col == sortCol {
				label = label + "▾"
			}
			formatted := fmt.Sprintf("%-*s", c.width, label)
			if c.col == sortCol {
				s = styleSortIndicator.Render(formatted)
			} else {
				s = styleTableHeader.Render(formatted)
			}
		}
		if i > 0 {
			parts = append(parts, " ")
		}
		parts = append(parts, s)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}
