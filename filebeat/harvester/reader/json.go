package reader

import (
	"encoding/json"
	"fmt"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

const (
	JsonErrorKey = "json_error"
)

type JSON struct {
	reader Reader
	cfg    *JSONConfig
}

// NewJSONReader creates a new reader that can decode JSON.
func NewJSON(r Reader, cfg *JSONConfig) *JSON {
	return &JSON{reader: r, cfg: cfg}
}

// decodeJSON unmarshals the text parameter into a MapStr and
// returns the new text column if one was requested.
func (r *JSON) decodeJSON(text []byte) ([]byte, common.MapStr) {
	var jsonFields common.MapStr
	err := json.Unmarshal(text, &jsonFields)
	if err != nil {
		logp.Err("Error decoding JSON: %v", err)
		if r.cfg.AddErrorKey {
			jsonFields = common.MapStr{JsonErrorKey: fmt.Sprintf("Error decoding JSON: %v", err)}
		}
		return text, jsonFields
	}

	if len(r.cfg.MessageKey) == 0 {
		return []byte(""), jsonFields
	}

	textValue, ok := jsonFields[r.cfg.MessageKey]
	if !ok {
		if r.cfg.AddErrorKey {
			jsonFields[JsonErrorKey] = fmt.Sprintf("Key '%s' not found", r.cfg.MessageKey)
		}
		return []byte(""), jsonFields
	}

	textString, ok := textValue.(string)
	if !ok {
		if r.cfg.AddErrorKey {
			jsonFields[JsonErrorKey] = fmt.Sprintf("Value of key '%s' is not a string", r.cfg.MessageKey)
		}
		return []byte(""), jsonFields
	}

	return []byte(textString), jsonFields
}

// Next decodes JSON and returns the filled Line object.
func (r *JSON) Next() (Message, error) {
	reader, err := r.reader.Next()
	if err != nil {
		return reader, err
	}
	reader.Content, reader.Fields = r.decodeJSON(reader.Content)
	return reader, nil
}
