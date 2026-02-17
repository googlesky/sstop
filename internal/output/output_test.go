package output

import (
	"bytes"
	"encoding/json"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/googlesky/sstop/internal/model"
)

func testSnapshot() model.Snapshot {
	return model.Snapshot{
		Timestamp: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
		Processes: []model.ProcessSummary{
			{
				PID:      1234,
				Name:     "firefox",
				Cmdline:  "/usr/bin/firefox",
				UpRate:   1024,
				DownRate: 2048,
				Connections: []model.Connection{
					{
						Proto:      model.ProtoTCP,
						SrcIP:      net.ParseIP("192.168.1.5"),
						SrcPort:    54321,
						DstIP:      net.ParseIP("142.250.80.46"),
						DstPort:    443,
						State:      model.StateEstablished,
						UpRate:     512,
						DownRate:   1024,
						Age:        5 * time.Minute,
						RemoteHost: "google.com",
					},
				},
				ConnCount: 1,
			},
			{
				PID:         22,
				Name:        "sshd",
				Cmdline:     "/usr/sbin/sshd",
				UpRate:      0,
				DownRate:    0,
				ListenPorts: []model.ListenPort{{Proto: model.ProtoTCP, IP: net.IPv4zero, Port: 22}},
				ListenCount: 1,
			},
		},
		Interfaces: []model.InterfaceStats{
			{Name: "eth0", BytesRecv: 100000, BytesSent: 50000, RecvRate: 2048, SendRate: 1024},
		},
		RemoteHosts: []model.RemoteHostSummary{
			{Host: "google.com", IP: net.ParseIP("142.250.80.46"), UpRate: 512, DownRate: 1024, ConnCount: 1, Processes: []string{"firefox"}},
		},
		ListenPorts: []model.ListenPortEntry{
			{Proto: model.ProtoTCP, IP: net.IPv4zero, Port: 22, PID: 22, Process: "sshd"},
		},
		TotalUp:   1024,
		TotalDown: 2048,
	}
}

func TestWriteJSON(t *testing.T) {
	snap := testSnapshot()
	var buf bytes.Buffer

	if err := WriteJSON(&buf, snap); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}

	// Must be valid JSON
	var decoded map[string]any
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, buf.String())
	}

	// Must end with newline (NDJSON)
	if !strings.HasSuffix(buf.String(), "\n") {
		t.Error("JSON output must end with newline")
	}

	// Check key fields exist
	if _, ok := decoded["timestamp"]; !ok {
		t.Error("missing timestamp field")
	}
	if _, ok := decoded["processes"]; !ok {
		t.Error("missing processes field")
	}
	if _, ok := decoded["total_up"]; !ok {
		t.Error("missing total_up field")
	}

	// Internal fields should NOT appear
	if _, ok := decoded["total_rate_history"]; ok {
		t.Error("total_rate_history should be excluded (json:\"-\")")
	}

	// Check process fields
	procs, ok := decoded["processes"].([]any)
	if !ok || len(procs) != 2 {
		t.Fatalf("expected 2 processes, got %v", decoded["processes"])
	}
	p0 := procs[0].(map[string]any)
	if p0["name"] != "firefox" {
		t.Errorf("expected process name firefox, got %v", p0["name"])
	}
	if p0["up_rate"] != float64(1024) {
		t.Errorf("expected up_rate 1024, got %v", p0["up_rate"])
	}
}

func TestWriteJSON_MultipleSnapshots(t *testing.T) {
	snap := testSnapshot()
	var buf bytes.Buffer

	for i := 0; i < 3; i++ {
		if err := WriteJSON(&buf, snap); err != nil {
			t.Fatalf("WriteJSON iteration %d: %v", i, err)
		}
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 NDJSON lines, got %d", len(lines))
	}

	for i, line := range lines {
		var decoded map[string]any
		if err := json.Unmarshal([]byte(line), &decoded); err != nil {
			t.Fatalf("line %d: invalid JSON: %v", i, err)
		}
	}
}

func TestCSVWriter(t *testing.T) {
	snap := testSnapshot()
	var buf bytes.Buffer

	w := NewCSVWriter(&buf)
	if err := w.Write(snap); err != nil {
		t.Fatalf("CSV Write: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	// 1 header + 2 process rows
	if len(lines) != 3 {
		t.Fatalf("expected 3 CSV lines (header + 2 rows), got %d: %v", len(lines), lines)
	}

	// Check header
	if lines[0] != "timestamp,pid,process,upload_bps,download_bps,connections,listen_ports" {
		t.Errorf("unexpected header: %s", lines[0])
	}

	// Check first data row
	if !strings.Contains(lines[1], "firefox") {
		t.Errorf("expected firefox in first data row: %s", lines[1])
	}
	if !strings.Contains(lines[1], "1234") {
		t.Errorf("expected PID 1234 in first data row: %s", lines[1])
	}
}

func TestCSVWriter_NoDoubleHeader(t *testing.T) {
	snap := testSnapshot()
	var buf bytes.Buffer

	w := NewCSVWriter(&buf)
	for i := 0; i < 3; i++ {
		if err := w.Write(snap); err != nil {
			t.Fatalf("CSV Write iteration %d: %v", i, err)
		}
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	// 1 header + 3*2 data rows = 7
	if len(lines) != 7 {
		t.Fatalf("expected 7 CSV lines, got %d", len(lines))
	}

	// Only first line should be header
	headerCount := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "timestamp,") {
			headerCount++
		}
	}
	if headerCount != 1 {
		t.Errorf("expected exactly 1 header, got %d", headerCount)
	}
}

func TestCSVWriter_EmptySnapshot(t *testing.T) {
	snap := model.Snapshot{Timestamp: time.Now()}
	var buf bytes.Buffer

	w := NewCSVWriter(&buf)
	if err := w.Write(snap); err != nil {
		t.Fatalf("CSV Write: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	// Only header, no data rows
	if len(lines) != 1 {
		t.Fatalf("expected 1 CSV line (header only), got %d", len(lines))
	}
}
