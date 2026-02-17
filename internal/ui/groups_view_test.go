package ui

import (
	"testing"

	"github.com/googlesky/sstop/internal/model"
)

func TestBuildGroups_MixedProcesses(t *testing.T) {
	procs := []model.ProcessSummary{
		{
			PID: 1, Name: "nginx", ContainerID: "abc123def456",
			UpRate: 100, DownRate: 200, ConnCount: 5,
		},
		{
			PID: 2, Name: "redis", ContainerID: "abc123def456",
			UpRate: 50, DownRate: 80, ConnCount: 3,
		},
		{
			PID: 3, Name: "sshd", ServiceName: "sshd.service",
			UpRate: 10, DownRate: 20, ConnCount: 1,
		},
		{
			PID: 4, Name: "firefox",
			UpRate: 500, DownRate: 1000, ConnCount: 20,
		},
	}

	groups := buildGroups(procs)

	if len(groups) != 3 {
		t.Fatalf("got %d groups, want 3", len(groups))
	}

	// Groups should be sorted by total rate descending.
	// "other" (user): 500+1000=1500
	// "abc123def456" (container): 100+200+50+80=430
	// "sshd.service" (systemd): 10+20=30

	if groups[0].Name != "other" || groups[0].Type != "user" {
		t.Errorf("groups[0] = {Name:%q, Type:%q}, want {Name:\"other\", Type:\"user\"}", groups[0].Name, groups[0].Type)
	}
	if groups[0].ProcCount != 1 {
		t.Errorf("groups[0].ProcCount = %d, want 1", groups[0].ProcCount)
	}
	if groups[0].UpRate != 500 {
		t.Errorf("groups[0].UpRate = %f, want 500", groups[0].UpRate)
	}
	if groups[0].DownRate != 1000 {
		t.Errorf("groups[0].DownRate = %f, want 1000", groups[0].DownRate)
	}
	if groups[0].ConnCount != 20 {
		t.Errorf("groups[0].ConnCount = %d, want 20", groups[0].ConnCount)
	}

	if groups[1].Name != "abc123def456" || groups[1].Type != "container" {
		t.Errorf("groups[1] = {Name:%q, Type:%q}, want {Name:\"abc123def456\", Type:\"container\"}", groups[1].Name, groups[1].Type)
	}
	if groups[1].ProcCount != 2 {
		t.Errorf("groups[1].ProcCount = %d, want 2", groups[1].ProcCount)
	}
	if groups[1].UpRate != 150 {
		t.Errorf("groups[1].UpRate = %f, want 150", groups[1].UpRate)
	}
	if groups[1].DownRate != 280 {
		t.Errorf("groups[1].DownRate = %f, want 280", groups[1].DownRate)
	}
	if groups[1].ConnCount != 8 {
		t.Errorf("groups[1].ConnCount = %d, want 8", groups[1].ConnCount)
	}

	if groups[2].Name != "sshd.service" || groups[2].Type != "systemd" {
		t.Errorf("groups[2] = {Name:%q, Type:%q}, want {Name:\"sshd.service\", Type:\"systemd\"}", groups[2].Name, groups[2].Type)
	}
	if groups[2].ProcCount != 1 {
		t.Errorf("groups[2].ProcCount = %d, want 1", groups[2].ProcCount)
	}
	if groups[2].ConnCount != 1 {
		t.Errorf("groups[2].ConnCount = %d, want 1", groups[2].ConnCount)
	}
}

func TestBuildGroups_EmptyProcessList(t *testing.T) {
	groups := buildGroups(nil)
	if len(groups) != 0 {
		t.Errorf("got %d groups for nil input, want 0", len(groups))
	}

	groups = buildGroups([]model.ProcessSummary{})
	if len(groups) != 0 {
		t.Errorf("got %d groups for empty slice, want 0", len(groups))
	}
}

func TestBuildGroups_Sorting(t *testing.T) {
	// Verify groups are sorted by total rate (up+down) descending
	procs := []model.ProcessSummary{
		{PID: 1, Name: "low", ServiceName: "low.service", UpRate: 1, DownRate: 1},
		{PID: 2, Name: "high", ServiceName: "high.service", UpRate: 1000, DownRate: 1000},
		{PID: 3, Name: "mid", ServiceName: "mid.service", UpRate: 100, DownRate: 100},
	}

	groups := buildGroups(procs)

	if len(groups) != 3 {
		t.Fatalf("got %d groups, want 3", len(groups))
	}

	expectedOrder := []string{"high.service", "mid.service", "low.service"}
	for i, expected := range expectedOrder {
		if groups[i].Name != expected {
			t.Errorf("groups[%d].Name = %q, want %q", i, groups[i].Name, expected)
		}
	}
}

func TestBuildGroups_AggregatesSameGroup(t *testing.T) {
	// Multiple processes in the same container should aggregate
	procs := []model.ProcessSummary{
		{PID: 1, Name: "worker-1", ContainerID: "container1", UpRate: 10, DownRate: 20, ConnCount: 2},
		{PID: 2, Name: "worker-2", ContainerID: "container1", UpRate: 30, DownRate: 40, ConnCount: 3},
		{PID: 3, Name: "worker-3", ContainerID: "container1", UpRate: 50, DownRate: 60, ConnCount: 5},
	}

	groups := buildGroups(procs)

	if len(groups) != 1 {
		t.Fatalf("got %d groups, want 1", len(groups))
	}

	g := groups[0]
	if g.ProcCount != 3 {
		t.Errorf("ProcCount = %d, want 3", g.ProcCount)
	}
	if g.UpRate != 90 {
		t.Errorf("UpRate = %f, want 90", g.UpRate)
	}
	if g.DownRate != 120 {
		t.Errorf("DownRate = %f, want 120", g.DownRate)
	}
	if g.ConnCount != 10 {
		t.Errorf("ConnCount = %d, want 10", g.ConnCount)
	}
}

func TestTruncateStr(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		maxLen int
		want   string
	}{
		{"shorter than max", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"needs truncation", "hello world", 8, "hello .."},
		{"maxLen 2", "hello", 2, "he"},
		{"maxLen 1", "hello", 1, "h"},
		{"maxLen 0", "hello", 0, ""},
		{"empty string", "", 5, ""},
		{"truncate to 3", "abcdef", 3, "a.."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateStr(tt.s, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateStr(%q, %d) = %q, want %q", tt.s, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestClassifyGroup_Container(t *testing.T) {
	proc := &model.ProcessSummary{
		PID:         1,
		Name:        "nginx",
		ContainerID: "abc123def456",
	}
	name, typ := classifyGroup(proc)
	if name != "abc123def456" {
		t.Errorf("name = %q, want %q", name, "abc123def456")
	}
	if typ != "container" {
		t.Errorf("typ = %q, want %q", typ, "container")
	}
}

func TestClassifyGroup_Systemd(t *testing.T) {
	proc := &model.ProcessSummary{
		PID:         2,
		Name:        "sshd",
		ServiceName: "sshd.service",
	}
	name, typ := classifyGroup(proc)
	if name != "sshd.service" {
		t.Errorf("name = %q, want %q", name, "sshd.service")
	}
	if typ != "systemd" {
		t.Errorf("typ = %q, want %q", typ, "systemd")
	}
}

func TestClassifyGroup_User(t *testing.T) {
	proc := &model.ProcessSummary{
		PID:  3,
		Name: "firefox",
	}
	name, typ := classifyGroup(proc)
	if name != "other" {
		t.Errorf("name = %q, want %q", name, "other")
	}
	if typ != "user" {
		t.Errorf("typ = %q, want %q", typ, "user")
	}
}

func TestClassifyGroup_ContainerPrecedence(t *testing.T) {
	// When both ContainerID and ServiceName are set, ContainerID takes precedence
	proc := &model.ProcessSummary{
		PID:         4,
		Name:        "app",
		ContainerID: "xyz789",
		ServiceName: "app.service",
	}
	name, typ := classifyGroup(proc)
	if name != "xyz789" {
		t.Errorf("name = %q, want %q (ContainerID should take precedence)", name, "xyz789")
	}
	if typ != "container" {
		t.Errorf("typ = %q, want %q", typ, "container")
	}
}
