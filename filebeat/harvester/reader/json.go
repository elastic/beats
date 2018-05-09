package reader

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/jsontransform"
	"github.com/elastic/beats/libbeat/logp"
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
	var jsonFields map[string]interface{}

	err := unmarshal(text, &jsonFields)
	if err != nil || jsonFields == nil {
		if !r.cfg.IgnoreDecodingError {
			logp.Err("Error decoding JSON: %v", err)
		}
		if r.cfg.AddErrorKey {
			jsonFields = common.MapStr{"error": createJSONError(fmt.Sprintf("Error decoding JSON: %v", err))}
		}
		return text, jsonFields
	}

	if len(r.cfg.MessageKey) == 0 {
		return []byte(""), jsonFields
	}

	textValue, ok := jsonFields[r.cfg.MessageKey]
	if !ok {
		if r.cfg.AddErrorKey {
			jsonFields["error"] = createJSONError(fmt.Sprintf("Key '%s' not found", r.cfg.MessageKey))
		}
		return []byte(""), jsonFields
	}

	textString, ok := textValue.(string)
	if !ok {
		if r.cfg.AddErrorKey {
			jsonFields["error"] = createJSONError(fmt.Sprintf("Value of key '%s' is not a string", r.cfg.MessageKey))
		}
		return []byte(""), jsonFields
	}

	return []byte(textString), jsonFields
}

// unmarshal is equivalent with json.Unmarshal but it converts numbers
// to int64 where possible, instead of using always float64.
func unmarshal(text []byte, fields *map[string]interface{}) error {
	dec := json.NewDecoder(bytes.NewReader(text))
	dec.UseNumber()
	err := dec.Decode(fields)
	if err != nil {
		return err
	}
	jsontransform.TransformNumbers(*fields)
	return nil
}

// Next decodes JSON and returns the filled Line object.
func (r *JSON) Next() (Message, error) {
	message, err := r.reader.Next()
	if err != nil {
		return message, err
	}

	var fields common.MapStr
	message.Content, fields = r.decodeJSON(message.Content)
	message.AddFields(common.MapStr{"json": fields})
	return message, nil
}

func createJSONError(message string) common.MapStr {
	return common.MapStr{"message": message, "type": "json"}
}

// MergeJSONFields writes the JSON fields in the event map,
// respecting the KeysUnderRoot and OverwriteKeys configuration options.
// If MessageKey is defined, the Text value from the event always
// takes precedence.
func MergeJSONFields(data common.MapStr, jsonFields common.MapStr, text *string, config JSONConfig) time.Time {
	// The message key might have been modified by multiline
	if len(config.MessageKey) > 0 && text != nil {
		jsonFields[config.MessageKey] = *text
	}

	// handle the case in which r.cfg.AddErrorKey is set and len(jsonFields) == 1
	// and only thing it contains is `error` key due to error in json decoding
	// which results in loss of message key in the main beat event
	if len(jsonFields) == 1 && jsonFields["error"] != nil {
		data["message"] = *text
	}

	if config.KeysUnderRoot {
		// Delete existing json key
		delete(data, "json")

		var ts time.Time
		if v, ok := data["@timestamp"]; ok {
			switch t := v.(type) {
			case time.Time:
				ts = t
			case common.Time:
				ts = time.Time(ts)
			}
			delete(data, "@timestamp")
		}
		event := &beat.Event{
			Timestamp: ts,
			Fields:    data,
		}
		jsontransform.WriteJSONKeys(event, jsonFields, config.OverwriteKeys)

		return event.Timestamp
	}
	return time.Time{}
}
