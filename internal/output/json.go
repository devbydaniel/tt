package output

import (
	"encoding/json"
	"io"
)

// WriteJSON writes data as indented JSON to the writer
func WriteJSON(w io.Writer, data any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}
