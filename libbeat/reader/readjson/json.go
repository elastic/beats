// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package readjson

import (
	"bytes"
	gojson "encoding/json"
	"fmt"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/jsontransform"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/reader"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// JSONReader parses JSON inputs
type JSONReader struct {
	reader reader.Reader
	cfg    *Config
	logger *logp.Logger
}

type JSONParser struct {
	JSONReader
	field, target string
}

// NewJSONReader creates a new reader that can decode JSON.
func NewJSONReader(r reader.Reader, cfg *Config) *JSONReader {
	return &JSONReader{
		reader: r,
		cfg:    cfg,
		logger: logp.NewLogger("reader_json"),
	}
}

func NewJSONParser(r reader.Reader, cfg *ParserConfig) *JSONParser {
	return &JSONParser{
		JSONReader{
			reader: r,
			cfg:    &cfg.Config,
			logger: logp.NewLogger("parser_json"),
		},
		cfg.Field,
		cfg.Target,
	}
}

// decodeJSON unmarshals the text parameter into a MapStr and
// returns the new text column if one was requested.
func (r *JSONReader) decode(text []byte) ([]byte, mapstr.M) {
	var jsonFields map[string]interface{}

	err := unmarshal(text, &jsonFields)
	if err != nil || jsonFields == nil {
		if !r.cfg.IgnoreDecodingError {
			r.logger.Errorf("Error decoding JSON: %v", err)
		}
		if r.cfg.AddErrorKey {
			jsonFields = mapstr.M{"error": createJSONError(fmt.Sprintf("Error decoding JSON: %v", err))}
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
	dec := gojson.NewDecoder(bytes.NewReader(text))
	dec.UseNumber()
	err := dec.Decode(fields)
	if err != nil {
		return err
	}
	jsontransform.TransformNumbers(*fields)
	return nil
}

// Next decodes JSON and returns the filled Line object.
func (r *JSONReader) Next() (reader.Message, error) {
	message, err := r.reader.Next()
	if err != nil {
		return message, err
	}

	var fields mapstr.M
	message.Content, fields = r.decode(message.Content)
	message.AddFields(mapstr.M{"json": fields})
	return message, nil
}

func (r *JSONReader) Close() error {
	return r.reader.Close()
}

func createJSONError(message string) mapstr.M {
	return mapstr.M{"message": message, "type": "json"}
}

// Next decodes JSON and returns the filled Line object.
func (p *JSONParser) Next() (reader.Message, error) {
	message, err := p.JSONReader.reader.Next()
	if err != nil {
		return message, err
	}

	var ok bool
	from := message.Content
	if p.field != "" {
		from, ok = message.Fields[p.field].([]byte)
		if !ok {
			return message, fmt.Errorf("cannot decode JSON message, missing key: %s", p.field)
		}
	}
	var jsonFields mapstr.M
	message.Content, jsonFields = p.JSONReader.decode(from)

	if len(jsonFields) == 0 {
		return message, err
	}

	// The message key might have been modified by multiline
	if len(p.cfg.MessageKey) > 0 && len(message.Content) > 0 {
		jsonFields[p.cfg.MessageKey] = string(message.Content)
	}

	// handle the case in which r.cfg.AddErrorKey is set and len(jsonFields) == 1
	// and only thing it contains is `error` key due to error in json decoding
	// which results in loss of message key in the main beat event
	if len(jsonFields) == 1 && jsonFields["error"] != nil {
		message.Fields["message"] = string(message.Content)
	}

	if key := p.JSONReader.cfg.DocumentID; key != "" {
		if tmp, err := jsonFields.GetValue(key); err == nil {
			if id, ok := tmp.(string); ok {
				jsonFields.Delete(key)

				if message.Meta == nil {
					message.Meta = mapstr.M{}
				}
				message.Meta["_id"] = id
			}
		}
	}

	if p.target == "" {
		event := &beat.Event{
			Timestamp: message.Ts,
			Meta:      message.Meta,
			Fields:    message.Fields,
		}
		jsontransform.WriteJSONKeys(event, jsonFields, p.JSONReader.cfg.ExpandKeys, p.JSONReader.cfg.OverwriteKeys, p.JSONReader.cfg.AddErrorKey)
		message.Ts = event.Timestamp
		message.Fields = event.Fields
		message.Meta = event.Meta
	} else {
		message.AddFields(mapstr.M{p.target: jsonFields})
	}

	return message, err
}

// MergeJSONFields writes the JSON fields in the event map,
// respecting the KeysUnderRoot, ExpandKeys, and OverwriteKeys configuration options.
// If MessageKey is defined, the Text value from the event always
// takes precedence.
func MergeJSONFields(data mapstr.M, jsonFields mapstr.M, text *string, config Config) (string, time.Time) {
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

	var id string
	if key := config.DocumentID; key != "" {
		if tmp, err := jsonFields.GetValue(key); err == nil {
			if v, ok := tmp.(string); ok {
				id = v
				jsonFields.Delete(key)
			}
		}
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
		jsontransform.WriteJSONKeys(event, jsonFields, config.ExpandKeys, config.OverwriteKeys, config.AddErrorKey)

		return id, event.Timestamp
	}
	return id, time.Time{}
}
