package output

import (
	"encoding/json"
	"io"

	"github.com/googlesky/sstop/internal/model"
)

// WriteJSON writes a single snapshot as one JSON line (NDJSON format).
func WriteJSON(w io.Writer, snap model.Snapshot) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(snap)
}
