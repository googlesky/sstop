package output

import (
	"encoding/csv"
	"fmt"
	"io"

	"github.com/googlesky/sstop/internal/model"
)

// CSVWriter writes snapshots as CSV rows.
type CSVWriter struct {
	w           *csv.Writer
	wroteHeader bool
}

// NewCSVWriter creates a new CSV writer.
func NewCSVWriter(w io.Writer) *CSVWriter {
	return &CSVWriter{w: csv.NewWriter(w)}
}

// Write writes one snapshot as CSV rows (one row per process).
func (c *CSVWriter) Write(snap model.Snapshot) error {
	if !c.wroteHeader {
		if err := c.w.Write([]string{
			"timestamp", "pid", "process", "upload_bps", "download_bps", "connections", "listen_ports",
		}); err != nil {
			return err
		}
		c.wroteHeader = true
	}

	ts := snap.Timestamp.Format("2006-01-02T15:04:05.000Z07:00")
	for _, p := range snap.Processes {
		if err := c.w.Write([]string{
			ts,
			fmt.Sprintf("%d", p.PID),
			p.Name,
			fmt.Sprintf("%.0f", p.UpRate),
			fmt.Sprintf("%.0f", p.DownRate),
			fmt.Sprintf("%d", p.ConnCount),
			fmt.Sprintf("%d", p.ListenCount),
		}); err != nil {
			return err
		}
	}
	c.w.Flush()
	return c.w.Error()
}
