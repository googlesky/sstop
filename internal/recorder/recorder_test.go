package recorder

import (
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/googlesky/sstop/internal/model"
)

func makeTestSnapshot(ts time.Time, nProcs int) model.Snapshot {
	var procs []model.ProcessSummary
	for i := 0; i < nProcs; i++ {
		procs = append(procs, model.ProcessSummary{
			PID:      uint32(1000 + i),
			Name:     "test-proc",
			UpRate:   float64(i * 100),
			DownRate: float64(i * 200),
			Connections: []model.Connection{
				{
					Proto:   model.ProtoTCP,
					SrcIP:   net.IPv4(127, 0, 0, 1),
					SrcPort: uint16(30000 + i),
					DstIP:   net.IPv4(8, 8, 8, 8),
					DstPort: 443,
					State:   model.StateEstablished,
					UpRate:  float64(i * 100),
				},
			},
		})
	}
	return model.Snapshot{
		Timestamp: ts,
		Processes: procs,
		TotalUp:   500.0,
		TotalDown: 1000.0,
		Interfaces: []model.InterfaceStats{
			{Name: "eth0", SendRate: 500, RecvRate: 1000},
		},
	}
}

func TestRecordAndPlaybackRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.ssrec")

	baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	// Record 5 snapshots
	rec, err := NewRecorder(path)
	if err != nil {
		t.Fatalf("NewRecorder: %v", err)
	}

	snaps := make([]model.Snapshot, 5)
	for i := 0; i < 5; i++ {
		snaps[i] = makeTestSnapshot(baseTime.Add(time.Duration(i)*time.Second), i+1)
		if err := rec.Write(snaps[i]); err != nil {
			t.Fatalf("Write[%d]: %v", i, err)
		}
	}
	if err := rec.Close(); err != nil {
		t.Fatalf("Close recorder: %v", err)
	}

	// Verify file exists and has content
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("recorded file is empty")
	}

	// Playback
	player, err := NewPlayer(path)
	if err != nil {
		t.Fatalf("NewPlayer: %v", err)
	}
	defer player.Close()

	if player.Len() != 5 {
		t.Fatalf("Len: got %d, want 5", player.Len())
	}

	// Fast playback (16x) to avoid slow test
	player.SetSpeed(16)

	ch := player.Play()
	var results []model.Snapshot
	for snap := range ch {
		results = append(results, snap)
	}

	if len(results) != 5 {
		t.Fatalf("got %d snapshots, want 5", len(results))
	}

	// Verify data fidelity
	for i, snap := range results {
		if len(snap.Processes) != i+1 {
			t.Errorf("snap[%d]: got %d procs, want %d", i, len(snap.Processes), i+1)
		}
		if snap.TotalUp != 500.0 {
			t.Errorf("snap[%d]: TotalUp got %f, want 500", i, snap.TotalUp)
		}
		if snap.TotalDown != 1000.0 {
			t.Errorf("snap[%d]: TotalDown got %f, want 1000", i, snap.TotalDown)
		}
		// Verify process data preserved
		for j, proc := range snap.Processes {
			if proc.PID != uint32(1000+j) {
				t.Errorf("snap[%d] proc[%d]: PID got %d, want %d", i, j, proc.PID, 1000+j)
			}
			if proc.Name != "test-proc" {
				t.Errorf("snap[%d] proc[%d]: Name got %q, want %q", i, j, proc.Name, "test-proc")
			}
			if len(proc.Connections) != 1 {
				t.Errorf("snap[%d] proc[%d]: got %d conns, want 1", i, j, len(proc.Connections))
			}
		}
		// Verify interface data preserved
		if len(snap.Interfaces) != 1 {
			t.Errorf("snap[%d]: got %d interfaces, want 1", i, len(snap.Interfaces))
		} else if snap.Interfaces[0].Name != "eth0" {
			t.Errorf("snap[%d]: iface name got %q, want %q", i, snap.Interfaces[0].Name, "eth0")
		}
	}
}

func TestRecordSession(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.ssrec")

	// Create a snapshot channel
	in := make(chan model.Snapshot, 3)

	out, _, err := RecordSession(in, path)
	if err != nil {
		t.Fatalf("RecordSession: %v", err)
	}

	baseTime := time.Date(2025, 6, 1, 10, 0, 0, 0, time.UTC)

	// Send snapshots
	for i := 0; i < 3; i++ {
		in <- makeTestSnapshot(baseTime.Add(time.Duration(i)*time.Second), 1)
	}
	close(in)

	// Drain output
	var results []model.Snapshot
	for snap := range out {
		results = append(results, snap)
	}

	if len(results) != 3 {
		t.Fatalf("got %d snapshots from output, want 3", len(results))
	}

	// Verify file was written and can be played back
	player, err := NewPlayer(path)
	if err != nil {
		t.Fatalf("NewPlayer: %v", err)
	}
	defer player.Close()

	if player.Len() != 3 {
		t.Fatalf("player Len: got %d, want 3", player.Len())
	}
}

func TestPlayerSpeedBounds(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "speed.ssrec")

	// Create a minimal recording
	rec, err := NewRecorder(path)
	if err != nil {
		t.Fatalf("NewRecorder: %v", err)
	}
	snap := makeTestSnapshot(time.Now(), 1)
	rec.Write(snap)
	rec.Close()

	player, err := NewPlayer(path)
	if err != nil {
		t.Fatalf("NewPlayer: %v", err)
	}

	// Default speed
	if player.Speed() != 1.0 {
		t.Errorf("default speed: got %f, want 1.0", player.Speed())
	}

	// Lower bound
	player.SetSpeed(0.1)
	if player.Speed() != 0.25 {
		t.Errorf("min speed: got %f, want 0.25", player.Speed())
	}

	// Upper bound
	player.SetSpeed(32)
	if player.Speed() != 16 {
		t.Errorf("max speed: got %f, want 16", player.Speed())
	}

	// Normal speed
	player.SetSpeed(4)
	if player.Speed() != 4 {
		t.Errorf("set speed 4: got %f, want 4", player.Speed())
	}
}

func TestPlayerPauseToggle(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pause.ssrec")

	rec, err := NewRecorder(path)
	if err != nil {
		t.Fatalf("NewRecorder: %v", err)
	}
	snap := makeTestSnapshot(time.Now(), 1)
	rec.Write(snap)
	rec.Close()

	player, err := NewPlayer(path)
	if err != nil {
		t.Fatalf("NewPlayer: %v", err)
	}

	if player.IsPaused() {
		t.Error("should not be paused initially")
	}

	player.TogglePause()
	if !player.IsPaused() {
		t.Error("should be paused after toggle")
	}

	player.TogglePause()
	if player.IsPaused() {
		t.Error("should not be paused after second toggle")
	}
}

func TestEmptyRecording(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.ssrec")

	// Create empty recording
	rec, err := NewRecorder(path)
	if err != nil {
		t.Fatalf("NewRecorder: %v", err)
	}
	rec.Close()

	player, err := NewPlayer(path)
	if err != nil {
		t.Fatalf("NewPlayer: %v", err)
	}

	if player.Len() != 0 {
		t.Errorf("empty recording Len: got %d, want 0", player.Len())
	}

	// Play should close channel immediately
	ch := player.Play()
	count := 0
	for range ch {
		count++
	}
	if count != 0 {
		t.Errorf("empty playback: got %d snapshots, want 0", count)
	}
}
