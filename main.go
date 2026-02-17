package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/googlesky/sstop/internal/collector"
	"github.com/googlesky/sstop/internal/model"
	"github.com/googlesky/sstop/internal/output"
	"github.com/googlesky/sstop/internal/platform"
	"github.com/googlesky/sstop/internal/recorder"
	"github.com/googlesky/sstop/internal/ui"
)

func main() {
	// Parse flags
	jsonFlag := flag.Bool("json", false, "Output JSONL (one JSON object per snapshot)")
	csvFlag := flag.Bool("csv", false, "Output CSV (header + rows per poll)")
	onceFlag := flag.Bool("once", false, "Single snapshot then exit")
	intervalFlag := flag.Duration("interval", 1*time.Second, "Poll interval (e.g. 2s, 500ms)")
	recordFlag := flag.String("record", "", "Record session to file (e.g. traffic.ssrec)")
	playbackFlag := flag.String("playback", "", "Playback a recorded session file")
	flag.Parse()

	if *jsonFlag && *csvFlag {
		fmt.Fprintln(os.Stderr, "error: --json and --csv are mutually exclusive")
		os.Exit(1)
	}

	// Playback mode — no platform/collector needed
	if *playbackFlag != "" {
		runPlayback(*playbackFlag)
		return
	}

	// Redirect log output to a file so it doesn't interfere with TUI
	logFile, err := os.CreateTemp("", "sstop-*.log")
	if err == nil {
		log.SetOutput(logFile)
		defer logFile.Close()
	}

	p, err := platform.NewPlatform()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to init platform: %v\n", err)
		os.Exit(1)
	}
	defer p.Close()

	interval := *intervalFlag
	if interval < 100*time.Millisecond {
		interval = 100 * time.Millisecond
	}

	c := collector.New(p, interval)
	snapCh := c.Start()
	defer c.Stop()

	// Non-interactive streaming mode
	if *jsonFlag || *csvFlag {
		runStreaming(snapCh, *jsonFlag, *onceFlag)
		return
	}

	// Record mode — wrap snapshot channel
	if *recordFlag != "" {
		recCh, _, err := recorder.RecordSession(snapCh, *recordFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to open record file: %v\n", err)
			os.Exit(1)
		}
		snapCh = recCh
	}

	// Smart detect the main outbound interface
	defaultIface := platform.DetectDefaultInterface()

	m := ui.New(snapCh)
	m.SetDefaultInterface(defaultIface)
	m.SetCollector(c)

	prog := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())

	if _, err := prog.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// Print exit summary
	stats := c.SessionStats()
	if summary := stats.Summary(); summary != "" {
		fmt.Print(summary)
	}
}

// runPlayback plays back a recorded session file.
func runPlayback(path string) {
	player, err := recorder.NewPlayer(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open playback file: %v\n", err)
		os.Exit(1)
	}
	defer player.Close()

	if player.Len() == 0 {
		fmt.Fprintln(os.Stderr, "recording is empty, nothing to play")
		os.Exit(1)
	}

	snapCh := player.Play()
	filename := filepath.Base(path)

	m := ui.New(snapCh)
	m.SetPlayback(player, filename)

	prog := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := prog.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// runStreaming handles --json / --csv non-interactive output.
func runStreaming(snapCh <-chan model.Snapshot, jsonMode bool, once bool) {
	// Need at least 2 polls for rate deltas: first poll gives no rates
	pollCount := 0

	var csvWriter *output.CSVWriter
	if !jsonMode {
		csvWriter = output.NewCSVWriter(os.Stdout)
	}

	for snap := range snapCh {
		pollCount++

		// Skip first poll — rates are all zero (no delta yet)
		if pollCount < 2 {
			continue
		}

		var err error
		if jsonMode {
			err = output.WriteJSON(os.Stdout, snap)
		} else {
			err = csvWriter.Write(snap)
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "write error: %v\n", err)
			os.Exit(1)
		}

		if once {
			return
		}
	}
}
