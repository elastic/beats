package reader

import (
	"bytes"
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
	var jsonFields map[string]interface{}

	err := unmarshal(text, &jsonFields)
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

// unmarshal is equivalent with json.Unmarshal but it converts numbers
// to int64 where possible, instead of using always float64.
func unmarshal(text []byte, fields *map[string]interface{}) error {
	dec := json.NewDecoder(bytes.NewReader(text))
	dec.UseNumber()
	err := dec.Decode(fields)
	if err != nil {
		return err
	}
	transformNumbersDict(*fields)
	return nil
}

// transformNumbersDict walks a json decoded tree an replaces json.Number
// with int64, float64, or string, in this order of preference (i.e. if it
// parses as an int, use int. if it parses as a float, use float. etc).
func transformNumbersDict(dict common.MapStr) {
	for k, v := range dict {
		switch vv := v.(type) {
		case json.Number:
			dict[k] = transformNumber(vv)
		case map[string]interface{}:
			transformNumbersDict(vv)
		case []interface{}:
			transformNumbersArray(vv)
		}
	}
}

func transformNumber(value json.Number) interface{} {
	i64, err := value.Int64()
	if err == nil {
		return i64
	}
	f64, err := value.Float64()
	if err == nil {
		return f64
	}
	return value.String()
}

func transformNumbersArray(arr []interface{}) {
	for i, v := range arr {
		switch vv := v.(type) {
		case json.Number:
			arr[i] = transformNumber(vv)
		case map[string]interface{}:
			transformNumbersDict(vv)
		case []interface{}:
			transformNumbersArray(vv)
		}
	}
}

// Next decodes JSON and returns the filled Line object.
func (r *JSON) Next() (Message, error) {
	message, err := r.reader.Next()
	if err != nil {
		return message, err
	}
	message.Content, message.Fields = r.decodeJSON(message.Content)
	return message, nil
}
