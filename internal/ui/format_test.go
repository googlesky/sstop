package ui

import (
	"testing"
	"time"
)

func TestFormatRateCompact_Width(t *testing.T) {
	// FormatRateCompact must ALWAYS return exactly 6 chars
	testCases := []float64{
		0,
		0.5,
		1,
		42,
		100,
		999,
		1023,
		1024,            // 1 K
		5 * 1024,        // 5 K
		9.9 * 1024,      // 9.9 K
		10 * 1024,       // 10 K
		100 * 1024,      // 100 K
		999 * 1024,      // 999 K
		1023 * 1024,     // 1023 K (close to M boundary)
		1024 * 1024,     // 1 M
		5 * 1024 * 1024, // 5 M
		9.9 * 1024 * 1024,
		10 * 1024 * 1024,
		100 * 1024 * 1024,
		999 * 1024 * 1024,
		1024 * 1024 * 1024,     // 1 G
		5 * 1024 * 1024 * 1024, // 5 G
		10 * 1024 * 1024 * 1024,
	}

	for _, bps := range testCases {
		result := FormatRateCompact(bps)
		if len(result) != 6 {
			t.Errorf("FormatRateCompact(%v) = %q (len=%d), want len=6", bps, result, len(result))
		}
	}
}

func TestFormatRateCompact_Values(t *testing.T) {
	tests := []struct {
		bps  float64
		want string
	}{
		{0, "   0 B"},
		{42, "  42 B"},
		{999, " 999 B"},
		{1024, "  1.0K"},
		{5 * 1024, "  5.0K"},
		{10 * 1024, "   10K"},
		{100 * 1024, "  100K"},
		{1024 * 1024, "  1.0M"},
		{10 * 1024 * 1024, "   10M"},
		{1024 * 1024 * 1024, "  1.0G"},
	}

	for _, tt := range tests {
		result := FormatRateCompact(tt.bps)
		if result != tt.want {
			t.Errorf("FormatRateCompact(%v) = %q, want %q", tt.bps, result, tt.want)
		}
	}
}

func TestFormatRateCompact_FuzzWidths(t *testing.T) {
	// Fuzz test: check a wide range of values all produce 6 chars
	for exp := 0.0; exp < 40; exp += 0.1 {
		bps := 1.0
		for i := 0.0; i < exp; i++ {
			bps *= 2
		}
		result := FormatRateCompact(bps)
		if len(result) != 6 {
			t.Errorf("FormatRateCompact(2^%.1f = %v) = %q (len=%d), want len=6",
				exp, bps, result, len(result))
		}
	}
}

func TestFormatRate(t *testing.T) {
	tests := []struct {
		bps  float64
		want string
	}{
		{0, "0 B/s"},
		{42, "42 B/s"},
		{1024, "1.0 KB/s"},
		{1024 * 1024, "1.0 MB/s"},
	}
	for _, tt := range tests {
		result := FormatRate(tt.bps)
		if result != tt.want {
			t.Errorf("FormatRate(%v) = %q, want %q", tt.bps, result, tt.want)
		}
	}
}

func TestSparkline(t *testing.T) {
	// Empty
	if s := Sparkline(nil, 5); s != "     " {
		t.Errorf("empty sparkline = %q, want 5 spaces", s)
	}

	// Single value
	s := Sparkline([]float64{100}, 5)
	if len([]rune(s)) != 5 {
		t.Errorf("single value sparkline width = %d, want 5", len([]rune(s)))
	}

	// Full
	vals := []float64{0, 25, 50, 75, 100}
	s = Sparkline(vals, 5)
	if len([]rune(s)) != 5 {
		t.Errorf("full sparkline width = %d, want 5", len([]rune(s)))
	}
}

func TestBandwidthBar(t *testing.T) {
	// Zero rate
	bar := BandwidthBar(0, 100, 5)
	if bar != "     " {
		t.Errorf("zero bar = %q, want 5 spaces", bar)
	}

	// Max rate
	bar = BandwidthBar(100, 100, 5)
	if bar != "█████" {
		t.Errorf("max bar = %q, want 5 full blocks", bar)
	}

	// Width consistency
	for r := 0.0; r <= 100; r += 1 {
		bar = BandwidthBar(r, 100, 5)
		if len([]rune(bar)) != 5 {
			t.Errorf("BandwidthBar(%v, 100, 5) width = %d, want 5", r, len([]rune(bar)))
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		s      string
		maxLen int
		want   string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "hell~"},
		{"", 5, ""},
		{"hi", 1, "~"},
	}
	for _, tt := range tests {
		result := Truncate(tt.s, tt.maxLen)
		if result != tt.want {
			t.Errorf("Truncate(%q, %d) = %q, want %q", tt.s, tt.maxLen, result, tt.want)
		}
	}
}

func TestFormatAge(t *testing.T) {
	tests := []struct {
		dur  time.Duration
		want string
	}{
		{5 * time.Second, "5s"},
		{65 * time.Second, "1m5s"},
		{3665 * time.Second, "1h1m"},
		{90000 * time.Second, "1d1h"},
	}
	for _, tt := range tests {
		result := FormatAge(tt.dur)
		if result != tt.want {
			t.Errorf("FormatAge(%v) = %q, want %q", tt.dur, result, tt.want)
		}
	}
}

func TestTrendArrow(t *testing.T) {
	// Not enough samples
	if a := TrendArrow([]float64{1, 2}); a != " " {
		t.Errorf("short history = %q, want space", a)
	}

	// Rising
	history := []float64{1, 1, 1, 1, 5, 5, 5, 5}
	if a := TrendArrow(history); a != "↑" {
		t.Errorf("rising = %q, want ↑", a)
	}

	// Falling
	history = []float64{5, 5, 5, 5, 1, 1, 1, 1}
	if a := TrendArrow(history); a != "↓" {
		t.Errorf("falling = %q, want ↓", a)
	}

	// Stable
	history = []float64{5, 5, 5, 5, 5, 5, 5, 5}
	if a := TrendArrow(history); a != "→" {
		t.Errorf("stable = %q, want →", a)
	}
}
