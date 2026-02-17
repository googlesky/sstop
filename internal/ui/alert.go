package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/googlesky/sstop/internal/model"
)

// alertOverlay manages bandwidth threshold alerts.
type alertOverlay struct {
	active         bool
	editing        bool
	input          textinput.Model
	threshold      float64 // bytes/sec, 0 = disabled
	alertTriggered map[uint32]bool // PIDs that have already triggered bell
	flashOn        bool // toggle for flash animation
}

func newAlertOverlay() alertOverlay {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Placeholder = "e.g. 1M, 500K, 10G"
	ti.CharLimit = 16
	return alertOverlay{
		input:          ti,
		alertTriggered: make(map[uint32]bool),
	}
}

func (a *alertOverlay) open() {
	a.active = true
	a.editing = true
	a.input.Focus()
	if a.threshold > 0 {
		a.input.SetValue(formatThreshold(a.threshold))
	}
}

func (a *alertOverlay) close() {
	a.active = false
	a.editing = false
	a.input.Blur()
}

func (a *alertOverlay) disable() {
	a.threshold = 0
	a.alertTriggered = make(map[uint32]bool)
	a.close()
}

func (a *alertOverlay) confirm() {
	val := strings.TrimSpace(a.input.Value())
	if val == "" || val == "0" {
		a.disable()
		return
	}
	parsed := parseSize(val)
	if parsed <= 0 {
		a.disable()
		return
	}
	a.threshold = parsed
	a.alertTriggered = make(map[uint32]bool)
	a.close()
}

// checkAlerts returns PIDs exceeding threshold and whether bell should ring.
func (a *alertOverlay) checkAlerts(procs []model.ProcessSummary) (exceeding []uint32, bell bool) {
	if a.threshold <= 0 {
		return nil, false
	}

	for _, p := range procs {
		total := p.UpRate + p.DownRate
		if total > a.threshold {
			exceeding = append(exceeding, p.PID)
			if !a.alertTriggered[p.PID] {
				a.alertTriggered[p.PID] = true
				bell = true
			}
		}
	}

	// Clean up: remove triggered PIDs that are no longer exceeding
	activeSet := make(map[uint32]bool)
	for _, pid := range exceeding {
		activeSet[pid] = true
	}
	for pid := range a.alertTriggered {
		if !activeSet[pid] {
			delete(a.alertTriggered, pid)
		}
	}

	return exceeding, bell
}

// isExceeding returns true if the PID is currently exceeding threshold.
func (a *alertOverlay) isExceeding(pid uint32) bool {
	if a.threshold <= 0 {
		return false
	}
	return a.alertTriggered[pid]
}

func (a *alertOverlay) render(width, height int) string {
	boxW := 48
	if boxW > width-4 {
		boxW = width - 4
	}

	title := styleSortIndicator.Render(" Set Bandwidth Alert ")
	content := styleDetailLabel.Render("Alert when any process exceeds:") + "\n\n"
	content += "  " + a.input.View() + " /s\n\n"
	content += styleDetailLabel.Render("  Enter to confirm, Esc to cancel")
	if a.threshold > 0 {
		content += "\n" + styleDetailLabel.Render("  Current: "+formatThreshold(a.threshold)+"/s")
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorAccent).
		Width(boxW).
		Padding(1, 2).
		Render(title + "\n\n" + content)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}

func (a *alertOverlay) update(msg tea.KeyMsg) tea.Cmd {
	if !a.editing {
		return nil
	}
	switch msg.String() {
	case "enter":
		a.confirm()
		return nil
	case "esc":
		a.close()
		return nil
	default:
		var cmd tea.Cmd
		a.input, cmd = a.input.Update(msg)
		return cmd
	}
}

// alertHeaderText returns the alert indicator for the header.
func (a *alertOverlay) alertHeaderText(procs []model.ProcessSummary) string {
	if a.threshold <= 0 {
		return ""
	}
	count := 0
	for _, p := range procs {
		if p.UpRate+p.DownRate > a.threshold {
			count++
		}
	}
	if count > 0 {
		return fmt.Sprintf(" ⚠ %d > %s/s ", count, formatThreshold(a.threshold))
	}
	return fmt.Sprintf(" ✓ < %s/s ", formatThreshold(a.threshold))
}

func formatThreshold(t float64) string {
	const (
		K = 1024.0
		M = K * 1024
		G = M * 1024
	)
	switch {
	case t >= G:
		v := t / G
		if v == float64(int(v)) {
			return strconv.Itoa(int(v)) + "G"
		}
		return fmt.Sprintf("%.1fG", v)
	case t >= M:
		v := t / M
		if v == float64(int(v)) {
			return strconv.Itoa(int(v)) + "M"
		}
		return fmt.Sprintf("%.1fM", v)
	case t >= K:
		v := t / K
		if v == float64(int(v)) {
			return strconv.Itoa(int(v)) + "K"
		}
		return fmt.Sprintf("%.1fK", v)
	default:
		return fmt.Sprintf("%.0f", t)
	}
}
