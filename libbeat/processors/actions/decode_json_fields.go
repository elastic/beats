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
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"

	jsoniter "github.com/json-iterator/go"

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

// djfAPI configures jsoniter to surface numbers as json.Number so that
// parseValue can convert them to int64/float64 inline without a second pass.
var djfAPI = jsoniter.Config{UseNumber: true}.Froze()

// djfIterState bundles a reusable iterator and scratch buffer.
// One instance is held in iterPool per concurrent caller; it must not be
// used across goroutines simultaneously.
type djfIterState struct {
	iter   *jsoniter.Iterator
	numBuf []byte // scratch buffer for number parsing; reset to [:0] before each number
}

type decodeJSONFields struct {
	fields        []string
	maxDepth      int
	expandKeys    bool
	overwriteKeys bool
	addErrorKey   bool
	processArray  bool
	documentID    string
	target        *string
	logger        *logp.Logger
	iterPool      sync.Pool // pools *djfIterState; safe for concurrent Run calls
}

type config struct {
	Fields        []string `config:"fields"`
	MaxDepth      int      `config:"max_depth" validate:"min=1"`
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
			checks.AllowedFields("fields", "max_depth", "overwrite_keys", "add_error_key", "process_array", "target", "when", "document_id", "expand_keys")))

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
		expandKeys:    config.ExpandKeys,
		overwriteKeys: config.OverwriteKeys,
		addErrorKey:   config.AddErrorKey,
		processArray:  config.ProcessArray,
		documentID:    config.DocumentID,
		target:        config.Target,
		logger:        logger,
		iterPool: sync.Pool{
			New: func() interface{} {
				return &djfIterState{
					iter:   jsoniter.NewIterator(djfAPI),
					numBuf: make([]byte, 0, 64),
				}
			},
		},
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

		var output interface{}
		err = f.unmarshal(f.maxDepth, text, &output, f.processArray)
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
			if dict, ok := output.(map[string]interface{}); ok {
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
				case map[string]interface{}:
					jsontransform.ExpandFields(f.logger, event, t, f.addErrorKey)
				default:
					errs = append(errs, "failed to expand keys")
				}
			}
			_, err = event.PutValue(target, output)
		} else {
			switch t := output.(type) {
			case map[string]interface{}:
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

// unmarshal decodes text as JSON and, when maxDepth > 1, recursively decodes
// any string values that look like JSON objects or arrays.
func (f *decodeJSONFields) unmarshal(maxDepth int, text string, fields *interface{}, processArray bool) error {
	if err := f.decodeJSON(text, fields); err != nil {
		return err
	}

	maxDepth--
	if maxDepth == 0 {
		return nil
	}

	tryUnmarshal := func(v interface{}) (interface{}, bool) {
		str, isString := v.(string)
		if !isString {
			return v, false
		} else if !isStructured(str) {
			return str, false
		}

		var tmp interface{}
		err := f.unmarshal(maxDepth, str, &tmp, processArray)
		if err != nil {
			return v, errors.Is(err, errProcessingSkipped)
		}

		return tmp, true
	}

	// try to deep unmarshal fields
	switch O := (*fields).(type) {
	case map[string]interface{}:
		for k, v := range O {
			if decoded, ok := tryUnmarshal(v); ok {
				O[k] = decoded
			}
		}
	// We want to process arrays here
	case []interface{}:
		if !processArray {
			return errProcessingSkipped
		}

		for i, v := range O {
			if decoded, ok := tryUnmarshal(v); ok {
				O[i] = decoded
			}
		}
	}
	return nil
}

// decodeJSON parses text as a single JSON value into *to, resolving numbers to
// int64 or float64 at parse time. It borrows a *djfIterState from the pool for
// the duration of the call; safe for concurrent use.
func (f *decodeJSONFields) decodeJSON(text string, to *interface{}) error {
	st := f.iterPool.Get().(*djfIterState)
	defer f.iterPool.Put(st)

	st.iter.ResetBytes([]byte(text))
	st.iter.Error = nil // ResetBytes does not clear prior errors
	*to = f.parseValue(st)
	if err := st.iter.Error; err != nil {
		st.iter.Error = nil
		return err
	}
	// Detect trailing content (multiple JSON elements in the input).
	//
	// WhatIsNext() peeks at the next token:
	//   - At clean EOF it calls loadMore(), which sets iter.Error = io.EOF and
	//     returns 0 → WhatIsNext returns InvalidValue with iter.Error = io.EOF.
	//   - For a trailing non-JSON char (e.g. "11:38:04") it reads the char,
	//     unreads it, and returns InvalidValue with iter.Error still nil.
	//   - For a second JSON value it returns a non-InvalidValue type.
	// Only the io.EOF case means we truly consumed all input.
	next := st.iter.WhatIsNext()
	parseErr := st.iter.Error
	st.iter.Error = nil
	if next != jsoniter.InvalidValue || !errors.Is(parseErr, io.EOF) {
		return errors.New("multiple json elements found")
	}
	return nil
}

// parseValue parses any JSON value from st.iter, resolving numbers inline.
func (f *decodeJSONFields) parseValue(st *djfIterState) interface{} {
	switch st.iter.WhatIsNext() {
	case jsoniter.StringValue:
		return st.iter.ReadString()
	case jsoniter.NumberValue:
		st.numBuf = st.numBuf[:0]
		st.numBuf = st.iter.SkipAndAppendBytes(st.numBuf)
		return djfParseNumber(st.numBuf)
	case jsoniter.BoolValue:
		return st.iter.ReadBool()
	case jsoniter.NilValue:
		st.iter.ReadNil()
		return nil
	case jsoniter.ObjectValue:
		nested := make(map[string]interface{}, 4)
		for field := st.iter.ReadObject(); field != ""; field = st.iter.ReadObject() {
			nested[field] = f.parseValue(st)
		}
		return nested
	case jsoniter.ArrayValue:
		arr := make([]interface{}, 0, 4)
		for st.iter.ReadArray() {
			arr = append(arr, f.parseValue(st))
		}
		return arr
	default:
		st.iter.Skip()
		return nil
	}
}

// djfParseNumber converts raw JSON number bytes to int64, float64, or string.
func djfParseNumber(b []byte) interface{} {
	if len(b) == 0 {
		return nil
	}
	for _, c := range b {
		if c == '.' || c == 'e' || c == 'E' {
			if fv, err := strconv.ParseFloat(string(b), 64); err == nil {
				return fv
			}
			return string(b)
		}
	}
	if i, err := strconv.ParseInt(string(b), 10, 64); err == nil {
		return i
	}
	if fv, err := strconv.ParseFloat(string(b), 64); err == nil {
		return fv
	}
	return string(b)
}

func (f *decodeJSONFields) String() string {
	return "decode_json_fields=" + strings.Join(f.fields, ", ")
}

func isStructured(s string) bool {
	s = strings.TrimSpace(s)
	end := len(s) - 1
	return end > 0 && ((s[0] == '[' && s[end] == ']') ||
		(s[0] == '{' && s[end] == '}'))
}
