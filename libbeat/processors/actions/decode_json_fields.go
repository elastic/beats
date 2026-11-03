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

package actions

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/beat/events"
	"github.com/elastic/beats/v7/libbeat/common/jsontransform"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/processors/checks"
	jsprocessor "github.com/elastic/beats/v7/libbeat/processors/script/javascript/module/processor/registry"
	cfg "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type decodeJSONFields struct {
	fields        []string
	maxDepth      int
	maxObjDepth   int
	expandKeys    bool
	overwriteKeys bool
	addErrorKey   bool
	processArray  bool
	documentID    string
	target        *string
	logger        *logp.Logger
}

type config struct {
	Fields        []string `config:"fields"`
	MaxDepth      int      `config:"max_depth" validate:"min=1"`
	MaxObjDepth   int      `config:"max_object_depth" validate:"min=0"` // Zero means no limit.
	ExpandKeys    bool     `config:"expand_keys"`
	OverwriteKeys bool     `config:"overwrite_keys"`
	AddErrorKey   bool     `config:"add_error_key"`
	ProcessArray  bool     `config:"process_array"`
	Target        *string  `config:"target"`
	DocumentID    string   `config:"document_id"`
}

var (
	defaultConfig = config{
		MaxDepth:     1,
		ProcessArray: false,
	}
	errProcessingSkipped = errors.New("processing skipped")
)

func init() {
	processors.RegisterPlugin("decode_json_fields",
		checks.ConfigChecked(NewDecodeJSONFields,
			checks.RequireFields("fields"),
			checks.AllowedFields("fields", "max_depth", "max_object_depth", "overwrite_keys", "add_error_key", "process_array", "target", "when", "document_id", "expand_keys")))

	jsprocessor.RegisterPlugin("DecodeJSONFields", NewDecodeJSONFields)
}

// NewDecodeJSONFields construct a new decode_json_fields processor.
func NewDecodeJSONFields(c *cfg.C, log *logp.Logger) (beat.Processor, error) {
	config := defaultConfig
	logger := log.Named("decode_json_fields")

	err := c.Unpack(&config)
	if err != nil {
		logger.Warn("Error unpacking config for decode_json_fields")
		return nil, fmt.Errorf("fail to unpack the decode_json_fields configuration: %w", err)
	}

	f := &decodeJSONFields{
		fields:        config.Fields,
		maxDepth:      config.MaxDepth,
		maxObjDepth:   config.MaxObjDepth,
		expandKeys:    config.ExpandKeys,
		overwriteKeys: config.OverwriteKeys,
		addErrorKey:   config.AddErrorKey,
		processArray:  config.ProcessArray,
		documentID:    config.DocumentID,
		target:        config.Target,
		logger:        logger,
	}
	return f, nil
}

func (f *decodeJSONFields) Run(event *beat.Event) (*beat.Event, error) {
	var errs []string

	for _, field := range f.fields {
		data, err := event.GetValue(field)
		if err != nil && !errors.Is(err, mapstr.ErrKeyNotFound) {
			f.logger.Debugf("Error trying to GetValue for field : %s in event : %v", field, event)
			errs = append(errs, err.Error())
			continue
		}

		text, ok := data.(string)
		if !ok {
			// ignore non string fields when unmarshaling
			continue
		}

		var output any
		err = unmarshal(f.maxDepth, text, &output, f.processArray, 1, f.maxObjDepth)
		if err != nil {
			f.logger.Debugf("Error trying to unmarshal %s", text)
			errs = append(errs, err.Error())
			event.SetErrorWithOption(fmt.Sprintf("parsing input as JSON: %s", err.Error()), f.addErrorKey, text, field)
			continue
		}

		target := field
		if f.target != nil {
			target = *f.target
		}

		var id string
		if key := f.documentID; key != "" {
			if dict, ok := output.(map[string]any); ok {
				if tmp, err := mapstr.M(dict).GetValue(key); err == nil {
					if v, ok := tmp.(string); ok {
						id = v
						_ = mapstr.M(dict).Delete(key)
					}
				}
			}
		}

		if target != "" {
			if f.expandKeys {
				switch t := output.(type) {
				case map[string]any:
					jsontransform.ExpandFields(f.logger, event, t, f.addErrorKey)
				default:
					errs = append(errs, "failed to expand keys")
				}
			}
			_, err = event.PutValue(target, output)
		} else {
			switch t := output.(type) {
			case map[string]any:
				jsontransform.WriteJSONKeys(event, t, f.expandKeys, f.overwriteKeys, f.addErrorKey)
			default:
				errs = append(errs, "failed to add target to root")
			}
		}

		if err != nil {
			f.logger.Debugf("Error trying to Put value %v for field : %s", output, field)
			errs = append(errs, err.Error())
			continue
		}

		if id != "" {
			if event.Meta == nil {
				event.Meta = mapstr.M{}
			}
			event.Meta[events.FieldMetaID] = id
		}
	}

	if len(errs) > 0 {
		return event, errors.New(strings.Join(errs, ", "))
	}
	return event, nil
}

// unmarshal decodes text into fields. The decoded value is at nesting level
// level, and if limit is greater than zero, objects and arrays that would be
// nested deeper than limit are kept as their original JSON text.
func unmarshal(maxDepth int, text string, fields *any, processArray bool, level, limit int) error {
	if err := decodeJSON(text, fields, level, limit); err != nil {
		return err
	}

	maxDepth--
	if maxDepth == 0 {
		return nil
	}

	tryUnmarshal := func(v any, level int) (any, bool) {
		str, isString := v.(string)
		if !isString {
			return v, false
		} else if !isStructured(str) {
			return str, false
		}
		if limit > 0 && level > limit {
			return v, false
		}

		var tmp any
		err := unmarshal(maxDepth, str, &tmp, processArray, level, limit)
		if err != nil {
			return v, errors.Is(err, errProcessingSkipped)
		}

		return tmp, true
	}

	// try to deep unmarshal fields
	switch O := (*fields).(type) {
	case map[string]any:
		for k, v := range O {
			if decoded, ok := tryUnmarshal(v, level+1); ok {
				O[k] = decoded
			}
		}
	// We want to process arrays here
	case []any:
		if !processArray {
			return errProcessingSkipped
		}

		for i, v := range O {
			if decoded, ok := tryUnmarshal(v, level); ok {
				O[i] = decoded
			}
		}
	}
	return nil
}

func decodeJSON(text string, to *any, level, limit int) error {
	dec := json.NewDecoder(strings.NewReader(text))
	dec.UseNumber()
	var err error
	if limit > 0 {
		*to, err = decodeLimited(dec, text, level, limit)
		if errors.Is(err, io.EOF) && strings.TrimSpace(text) != "" {
			// Token reports truncated input as io.EOF where Decode
			// reports io.ErrUnexpectedEOF.
			err = io.ErrUnexpectedEOF
		}
	} else {
		err = dec.Decode(to)
	}

	if err != nil {
		return err
	}

	if dec.More() {
		return errors.New("multiple json elements found")
	}

	if _, err := dec.Token(); err != nil && !errors.Is(err, io.EOF) {
		return err
	}

	switch O := (*to).(type) {
	case map[string]any:
		jsontransform.TransformNumbers(O)
	}
	return nil
}

// decodeLimited reads the next value from dec, which is decoding text. The
// value is at nesting level level. Objects and arrays that exceed limit are
// returned as the string of their original text, without being decoded.
// Elements of an array are at the same level as the array, since arrays do
// not add a level to field paths.
//
// TODO: Rewrite using encoding/json/jsontext when encoding/json/v2 is
// available without GOEXPERIMENT. Decoder.ReadValue returns the raw text of a
// value, which removes the need for skipContainer and the offset arithmetic.
func decodeLimited(dec *json.Decoder, text string, level, limit int) (any, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	delim, ok := tok.(json.Delim)
	if !ok {
		return tok, nil
	}

	if level > limit {
		// The offset is just after the opening delimiter.
		start := dec.InputOffset() - 1
		err := skipContainer(dec)
		if err != nil {
			return nil, err
		}
		return text[start:dec.InputOffset()], nil
	}

	switch delim {
	case '{':
		obj := make(map[string]any)
		for dec.More() {
			tok, err := dec.Token()
			if err != nil {
				return nil, err
			}
			key, ok := tok.(string)
			if !ok {
				return nil, fmt.Errorf("unexpected object key %v", tok)
			}
			obj[key], err = decodeLimited(dec, text, level+1, limit)
			if err != nil {
				return nil, err
			}
		}
		_, err = dec.Token()
		return obj, err
	case '[':
		arr := make([]any, 0)
		for dec.More() {
			v, err := decodeLimited(dec, text, level, limit)
			if err != nil {
				return nil, err
			}
			arr = append(arr, v)
		}
		_, err = dec.Token()
		return arr, err
	}
	return nil, fmt.Errorf("unexpected delimiter %v", delim)
}

// skipContainer reads tokens from dec up to and including the end of the
// object or array whose opening delimiter was the last token read.
func skipContainer(dec *json.Decoder) error {
	for depth := 1; depth > 0; {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		switch tok {
		case json.Delim('{'), json.Delim('['):
			depth++
		case json.Delim('}'), json.Delim(']'):
			depth--
		}
	}
	return nil
}

func (f decodeJSONFields) String() string {
	return "decode_json_fields=" + strings.Join(f.fields, ", ")
}

func isStructured(s string) bool {
	s = strings.TrimSpace(s)
	end := len(s) - 1
	return end > 0 && ((s[0] == '[' && s[end] == ']') ||
		(s[0] == '{' && s[end] == '}'))
}
