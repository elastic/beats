package processor

import (
	"encoding/json"
	"fmt"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

const (
	JsonErrorKey = "json_error"
)

type JSONProcessor struct {
	reader LineProcessor
	cfg    *JSONConfig
}

type JSONConfig struct {
	MessageKey    string `config:"message_key"`
	KeysUnderRoot bool   `config:"keys_under_root"`
	OverwriteKeys bool   `config:"overwrite_keys"`
	AddErrorKey   bool   `config:"add_error_key"`
}

// NewJSONProcessor creates a new processor that can decode JSON.
func NewJSONProcessor(in LineProcessor, cfg *JSONConfig) *JSONProcessor {
	return &JSONProcessor{reader: in, cfg: cfg}
}

// decodeJSON unmarshals the text parameter into a MapStr and
// returns the new text column if one was requested.
func (p *JSONProcessor) decodeJSON(text []byte) ([]byte, common.MapStr) {
	var jsonFields common.MapStr
	err := json.Unmarshal(text, &jsonFields)
	if err != nil {
		logp.Err("Error decoding JSON: %v", err)
		if p.cfg.AddErrorKey {
			jsonFields = common.MapStr{JsonErrorKey: fmt.Sprintf("Error decoding JSON: %v", err)}
		}
		return text, jsonFields
	}

	if len(p.cfg.MessageKey) == 0 {
		return []byte(""), jsonFields
	}

	textValue, ok := jsonFields[p.cfg.MessageKey]
	if !ok {
		if p.cfg.AddErrorKey {
			jsonFields[JsonErrorKey] = fmt.Sprintf("Key '%s' not found", p.cfg.MessageKey)
		}
		return []byte(""), jsonFields
	}

	textString, ok := textValue.(string)
	if !ok {
		if p.cfg.AddErrorKey {
			jsonFields[JsonErrorKey] = fmt.Sprintf("Value of key '%s' is not a string", p.cfg.MessageKey)
		}
		return []byte(""), jsonFields
	}

	return []byte(textString), jsonFields
}

// Next decodes JSON and returns the filled Line object.
func (p *JSONProcessor) Next() (Line, error) {
	line, err := p.reader.Next()
	if err != nil {
		return line, err
	}
	line.Content, line.Fields = p.decodeJSON(line.Content)
	return line, nil
}
