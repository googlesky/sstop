package ui

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// FormatBytes formats byte count to human-readable string.
func FormatBytes(b uint64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)
	switch {
	case b >= TB:
		return fmt.Sprintf("%.1f TB", float64(b)/float64(TB))
	case b >= GB:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(GB))
	case b >= MB:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(MB))
	case b >= KB:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(KB))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

// FormatRate formats a bytes/sec rate to human-readable string with /s suffix.
func FormatRate(bps float64) string {
	if bps < 0 {
		bps = 0
	}
	const (
		KB = 1024.0
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case bps >= GB:
		return fmt.Sprintf("%.1f GB/s", bps/GB)
	case bps >= MB:
		return fmt.Sprintf("%.1f MB/s", bps/MB)
	case bps >= KB:
		return fmt.Sprintf("%.1f KB/s", bps/KB)
	case bps >= 1:
		return fmt.Sprintf("%.0f B/s", bps)
	default:
		return "0 B/s"
	}
}

// FormatRateCompact formats a bytes/sec rate to a fixed-width string (always 6 chars).
// Uses compact units. Column headers already show "/s", so it's omitted here.
func FormatRateCompact(bps float64) string {
	const (
		K = 1024.0
		M = K * 1024
		G = M * 1024
	)
	switch {
	case bps < 1:
		return "   0 B"
	case bps < K:
		return fmt.Sprintf("%4.0f B", bps)
	case bps < 10*K:
		return fmt.Sprintf("%5.1fK", bps/K)
	case bps < M:
		return fmt.Sprintf("%5.0fK", bps/K)
	case bps < 10*M:
		return fmt.Sprintf("%5.1fM", bps/M)
	case bps < G:
		return fmt.Sprintf("%5.0fM", bps/M)
	case bps < 10*G:
		return fmt.Sprintf("%5.1fG", bps/G)
	default:
		return fmt.Sprintf("%5.0fG", bps/G)
	}
}

// Sparkline renders a slice of float64 values as a sparkline using Unicode blocks.
// The width parameter controls how many characters to output.
// Values are scaled relative to the maximum value in the slice.
func Sparkline(values []float64, width int) string {
	if width <= 0 || len(values) == 0 {
		return strings.Repeat(" ", width)
	}

	blocks := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

	// Use only the last `width` values
	if len(values) > width {
		values = values[len(values)-width:]
	}

	// Find max for scaling
	max := 0.0
	for _, v := range values {
		if v > max {
			max = v
		}
	}

	result := make([]rune, width)
	// Pad left with spaces if fewer values than width
	pad := width - len(values)
	for i := 0; i < pad; i++ {
		result[i] = ' '
	}

	for i, v := range values {
		if max <= 0 || v <= 0 {
			result[pad+i] = ' '
			continue
		}
		level := int(v / max * float64(len(blocks)-1))
		if level >= len(blocks) {
			level = len(blocks) - 1
		}
		result[pad+i] = blocks[level]
	}

	return string(result)
}

// BandwidthBar renders a proportional bar using Unicode block characters.
// rate is the current value, maxRate is the maximum value for scaling.
// width is the total character width of the bar output.
func BandwidthBar(rate, maxRate float64, width int) string {
	if width <= 0 {
		return ""
	}
	if maxRate <= 0 || rate <= 0 {
		return strings.Repeat(" ", width)
	}

	// Sub-block characters for 1/8th precision
	subBlocks := []rune{' ', '▏', '▎', '▍', '▌', '▋', '▊', '▉', '█'}

	ratio := rate / maxRate
	if ratio > 1.0 {
		ratio = 1.0
	}

	// Total fill in 1/8th units
	fillUnits := ratio * float64(width) * 8.0
	fullBlocks := int(fillUnits) / 8
	partialIdx := int(fillUnits) % 8

	result := make([]rune, width)
	for i := 0; i < width; i++ {
		if i < fullBlocks {
			result[i] = '█'
		} else if i == fullBlocks && partialIdx > 0 {
			result[i] = subBlocks[partialIdx]
		} else {
			result[i] = ' '
		}
	}
	return string(result)
}

// TrendArrow analyzes rate history and returns a directional arrow.
// Compares average of last 4 samples vs previous 4 samples.
func TrendArrow(history []float64) string {
	if len(history) < 4 {
		return " "
	}

	n := len(history)
	// Recent 4 samples
	recent := 0.0
	recentCount := 4
	if n < 8 {
		recentCount = n / 2
	}
	for i := n - recentCount; i < n; i++ {
		recent += history[i]
	}
	recent /= float64(recentCount)

	// Previous samples
	prevStart := n - recentCount*2
	if prevStart < 0 {
		prevStart = 0
	}
	prevCount := n - recentCount - prevStart
	if prevCount <= 0 {
		return " "
	}
	prev := 0.0
	for i := prevStart; i < prevStart+prevCount; i++ {
		prev += history[i]
	}
	prev /= float64(prevCount)

	// Determine trend
	if prev == 0 && recent == 0 {
		return " "
	}
	if prev == 0 {
		return "↑"
	}
	change := (recent - prev) / prev
	if change > 0.2 {
		return "↑"
	}
	if change < -0.2 {
		return "↓"
	}
	return "→"
}

// FormatAge formats a duration to a compact human-readable string.
func FormatAge(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	secs := int(d.Seconds())
	switch {
	case secs < 60:
		return fmt.Sprintf("%ds", secs)
	case secs < 3600:
		return fmt.Sprintf("%dm%ds", secs/60, secs%60)
	case secs < 86400:
		h := secs / 3600
		m := (secs % 3600) / 60
		return fmt.Sprintf("%dh%dm", h, m)
	default:
		d := secs / 86400
		h := (secs % 86400) / 3600
		return fmt.Sprintf("%dd%dh", d, h)
	}
}

// lerpValue linearly interpolates between a and b.
func lerpValue(a, b, t float64) float64 {
	return a + (b-a)*t
}

// clamp01 clamps a float64 to [0, 1].
func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// hslToRGB converts HSL (h: 0-360, s: 0-1, l: 0-1) to RGB (0-255 each).
func hslToRGB(h, s, l float64) (uint8, uint8, uint8) {
	c := (1 - math.Abs(2*l-1)) * s
	hp := h / 60.0
	x := c * (1 - math.Abs(math.Mod(hp, 2)-1))
	m := l - c/2

	var r, g, b float64
	switch {
	case hp < 1:
		r, g, b = c, x, 0
	case hp < 2:
		r, g, b = x, c, 0
	case hp < 3:
		r, g, b = 0, c, x
	case hp < 4:
		r, g, b = 0, x, c
	case hp < 5:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}

	return uint8((r + m) * 255), uint8((g + m) * 255), uint8((b + m) * 255)
}

// Truncate truncates a string to maxLen, adding "~" if truncated.
func Truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 1 {
		return "~"
	}
	return string(runes[:maxLen-1]) + "~"
}
