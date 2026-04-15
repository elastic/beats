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
	"fmt"
	"time"
	"unsafe"

	sonicDecoder "github.com/bytedance/sonic/decoder"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/jsontransform"
	"github.com/elastic/beats/v7/libbeat/reader"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// JSONReader parses JSON inputs
type JSONReader struct {
	reader reader.Reader
	cfg    *Config
	logger *logp.Logger
	dec    *sonicDecoder.Decoder // reused across calls; lazily initialised
}

type JSONParser struct {
	JSONReader
	field, target string
}

func newDecoder() *sonicDecoder.Decoder {
	dec := sonicDecoder.NewDecoder("")
	dec.UseNumber()
	dec.CopyString() // prevent decoded strings from aliasing the input buffer
	return dec
}

// NewJSONReader creates a new reader that can decode JSON.
func NewJSONReader(r reader.Reader, cfg *Config, logger *logp.Logger) *JSONReader {
	return &JSONReader{
		reader: r,
		cfg:    cfg,
		logger: logger.Named("reader_json"),
		dec:    newDecoder(),
	}
}

func NewJSONParser(r reader.Reader, cfg *ParserConfig, logger *logp.Logger) *JSONParser {
	return &JSONParser{
		JSONReader{
			reader: r,
			cfg:    &cfg.Config,
			logger: logger.Named("parser_json"),
			dec:    newDecoder(),
		},
		cfg.Field,
		cfg.Target,
	}
}

// decode unmarshals text as a JSON object into a MapStr and returns the new
// text column if MessageKey is configured. It reuses r.dec across calls to
// avoid per-line allocations; lazy init handles zero-value JSONReader in tests.
func (r *JSONReader) decode(text []byte) ([]byte, mapstr.M) {
	if r.dec == nil {
		r.dec = newDecoder()
	}
	var jsonFields map[string]interface{}
	r.dec.Reset(unsafe.String(unsafe.SliceData(text), len(text))) //nolint:gosec // G103: aliasing own slice for zero-copy reset
	err := r.dec.Decode(&jsonFields)
	if err != nil || jsonFields == nil {
		if !r.cfg.IgnoreDecodingError {
			r.logger.Errorf("Error decoding JSON: %v", err)
		}
		if r.cfg.AddErrorKey {
			return text, mapstr.M{"error": createJSONError(fmt.Sprintf("Error decoding JSON: %v", err))}
		}
		return text, nil
	}
	jsontransform.TransformNumbers(jsonFields)

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

// unmarshal parses text as a JSON object, converting numbers to int64 or
// float64. It creates a one-shot decoder; callers that parse many lines should
// use a JSONReader which reuses its decoder across calls.
func unmarshal(text []byte, fields *map[string]interface{}) error {
	dec := sonicDecoder.NewDecoder(unsafe.String(unsafe.SliceData(text), len(text))) //nolint:gosec
	dec.UseNumber()
	if err := dec.Decode(fields); err != nil {
		return err
	}
	if *fields != nil {
		jsontransform.TransformNumbers(*fields)
	}
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
		fields := mapstr.M{}
		fields.Put(p.target, jsonFields)
		message.AddFields(fields)
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
