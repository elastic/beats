package common

import (
	"bytes"
	"encoding/json"

	"fmt"
)

// JSONEncode encodes the given interface to JSON
func JSONEncode(data interface{}, pretty bool) ([]byte, error) {

	buffer := &bytes.Buffer{}
	enc := json.NewEncoder(buffer)
	enc.SetEscapeHTML(true)

	enc.SetIndent("", "")
	if pretty {
		enc.SetIndent("", "  ")
	}

	err := enc.Encode(data)
	if err != nil {
		return nil, fmt.Errorf("failed to convert the data to JSON (%v): %#v", err, data)
	}

	return buffer.Bytes(), nil
}
