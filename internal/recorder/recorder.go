package recorder

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"github.com/googlesky/sstop/internal/model"
)

// record wraps a snapshot with its timestamp for recording.
type record struct {
	Timestamp time.Time      `json:"ts"`
	Snapshot  model.Snapshot `json:"snap"`
}

// Recorder writes snapshots to a gzipped JSONL file.
type Recorder struct {
	mu   sync.Mutex
	file *os.File
	gz   *gzip.Writer
	enc  *json.Encoder
}

// NewRecorder creates a new recorder writing to the given file path.
func NewRecorder(path string) (*Recorder, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	gz := gzip.NewWriter(f)
	enc := json.NewEncoder(gz)
	enc.SetEscapeHTML(false)
	return &Recorder{file: f, gz: gz, enc: enc}, nil
}

// Write records a single snapshot.
func (r *Recorder) Write(snap model.Snapshot) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.enc.Encode(record{
		Timestamp: snap.Timestamp,
		Snapshot:  snap,
	})
}

// Close flushes and closes the recorder.
func (r *Recorder) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.gz.Close(); err != nil {
		r.file.Close()
		return err
	}
	return r.file.Close()
}

// RecordSession wraps a snapshot channel, recording all snapshots while passing them through.
func RecordSession(snapCh <-chan model.Snapshot, path string) (<-chan model.Snapshot, *Recorder, error) {
	rec, err := NewRecorder(path)
	if err != nil {
		return nil, nil, err
	}

	out := make(chan model.Snapshot, 1)
	go func() {
		defer close(out)
		defer rec.Close()
		for snap := range snapCh {
			if err := rec.Write(snap); err != nil {
				log.Printf("recorder: write error: %v", err)
			}
			select {
			case out <- snap:
			default:
				select {
				case <-out:
				default:
				}
				out <- snap
			}
		}
	}()

	return out, rec, nil
}

// Player reads recorded snapshots from a gzipped JSONL file.
type Player struct {
	file    *os.File
	gz      io.ReadCloser
	dec     *json.Decoder
	records []record
	idx     int

	mu     sync.Mutex
	speed  float64 // playback speed multiplier
	paused bool
}

// NewPlayer opens a recording file for playback.
func NewPlayer(path string) (*Player, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	gz, err := gzip.NewReader(f)
	if err != nil {
		f.Close()
		return nil, err
	}

	// Read all records into memory
	dec := json.NewDecoder(gz)
	var records []record
	for {
		var rec record
		if err := dec.Decode(&rec); err != nil {
			break
		}
		records = append(records, rec)
	}

	gz.Close()
	f.Close()

	return &Player{
		records: records,
		speed:   1.0,
	}, nil
}

// Play feeds snapshots to a channel at the original recording speed.
func (p *Player) Play() <-chan model.Snapshot {
	ch := make(chan model.Snapshot, 1)

	go func() {
		defer close(ch)

		for i := 0; i < len(p.records); i++ {
			for p.isPaused() {
				time.Sleep(100 * time.Millisecond)
			}

			snap := p.records[i].Snapshot
			snap.Timestamp = time.Now()
			ch <- snap

			// Wait for the delta between this and next snapshot
			if i+1 < len(p.records) {
				delta := p.records[i+1].Timestamp.Sub(p.records[i].Timestamp)
				speed := p.getSpeed()
				if delta > 0 && speed > 0 {
					time.Sleep(time.Duration(float64(delta) / speed))
				}
			}
		}
	}()

	return ch
}

// isPaused is the goroutine-safe internal reader for paused state.
func (p *Player) isPaused() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.paused
}

// getSpeed is the goroutine-safe internal reader for speed.
func (p *Player) getSpeed() float64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.speed
}

// SetSpeed sets the playback speed multiplier.
func (p *Player) SetSpeed(s float64) {
	if s < 0.25 {
		s = 0.25
	}
	if s > 16 {
		s = 16
	}
	p.mu.Lock()
	p.speed = s
	p.mu.Unlock()
}

// Speed returns current playback speed.
func (p *Player) Speed() float64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.speed
}

// TogglePause toggles pause state.
func (p *Player) TogglePause() {
	p.mu.Lock()
	p.paused = !p.paused
	p.mu.Unlock()
}

// IsPaused returns whether playback is paused.
func (p *Player) IsPaused() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.paused
}

// Len returns the number of recorded snapshots.
func (p *Player) Len() int {
	return len(p.records)
}

// Close releases resources.
func (p *Player) Close() error {
	return nil
}
