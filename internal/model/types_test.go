package model

import (
	"strings"
	"testing"
	"time"
)

func TestSessionStatsSummary(t *testing.T) {
	stats := SessionStats{
		Duration:  5*time.Minute + 32*time.Second,
		TotalUp:   149422080, // ~142.5 MB
		TotalDown: 1288490189, // ~1.2 GB
		TopProcess: []ProcessCumulative{
			{PID: 1, Name: "firefox", BytesUp: 47395430, BytesDown: 933232640},
			{PID: 2, Name: "curl", BytesUp: 96558285, BytesDown: 327680000},
		},
	}

	summary := stats.Summary()

	if !strings.Contains(summary, "5m32s") {
		t.Errorf("expected duration 5m32s in summary:\n%s", summary)
	}
	if !strings.Contains(summary, "142.") {
		t.Errorf("expected ~142 MB total up in summary:\n%s", summary)
	}
	if !strings.Contains(summary, "1.2 GB") {
		t.Errorf("expected 1.2 GB total down in summary:\n%s", summary)
	}
	if !strings.Contains(summary, "firefox") {
		t.Errorf("expected firefox in top processes:\n%s", summary)
	}
	if !strings.Contains(summary, "curl") {
		t.Errorf("expected curl in top processes:\n%s", summary)
	}
}

func TestSessionStatsSummaryEmpty(t *testing.T) {
	stats := SessionStats{}
	summary := stats.Summary()
	if summary != "" {
		t.Errorf("expected empty summary for zero stats, got: %s", summary)
	}
}

func TestFmtBytes(t *testing.T) {
	tests := []struct {
		input    uint64
		expected string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
		{1099511627776, "1.0 TB"},
	}
	for _, tt := range tests {
		got := fmtBytes(tt.input)
		if got != tt.expected {
			t.Errorf("fmtBytes(%d) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
