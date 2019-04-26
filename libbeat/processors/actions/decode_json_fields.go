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
	"fmt"
	"io"
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/jsontransform"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/processors"
	"github.com/elastic/beats/libbeat/processors/checks"
)

type decodeJSONFields struct {
	fields        []string
	maxDepth      int
	overwriteKeys bool
	processArray  bool
	target        *string
}

type config struct {
	Fields        []string `config:"fields"`
	MaxDepth      int      `config:"max_depth" validate:"min=1"`
	OverwriteKeys bool     `config:"overwrite_keys"`
	ProcessArray  bool     `config:"process_array"`
	Target        *string  `config:"target"`
}

var (
	defaultConfig = config{
		MaxDepth:     1,
		ProcessArray: false,
	}
	errProcessingSkipped = errors.New("processing skipped")
)

var debug = logp.MakeDebug("filters")

func init() {
	processors.RegisterPlugin("decode_json_fields",
		checks.ConfigChecked(NewDecodeJSONFields,
			checks.RequireFields("fields"),
			checks.AllowedFields("fields", "max_depth", "overwrite_keys", "process_array", "target", "when")))
}

// NewDecodeJSONFields construct a new decode_json_fields processor.
func NewDecodeJSONFields(c *common.Config) (processors.Processor, error) {
	config := defaultConfig

	err := c.Unpack(&config)
	if err != nil {
		logp.Warn("Error unpacking config for decode_json_fields")
		return nil, fmt.Errorf("fail to unpack the decode_json_fields configuration: %s", err)
	}

	f := &decodeJSONFields{fields: config.Fields, maxDepth: config.MaxDepth, overwriteKeys: config.OverwriteKeys, processArray: config.ProcessArray, target: config.Target}
	return f, nil
}

func (f *decodeJSONFields) Run(event *beat.Event) (*beat.Event, error) {
	var errs []string

	for _, field := range f.fields {
		data, err := event.GetValue(field)
		if err != nil && errors.Cause(err) != common.ErrKeyNotFound {
			debug("Error trying to GetValue for field : %s in event : %v", field, event)
			errs = append(errs, err.Error())
			continue
		}

		text, ok := data.(string)
		if !ok {
			// ignore non string fields when unmarshaling
			continue
		}

		var output interface{}
		err = unmarshal(f.maxDepth, text, &output, f.processArray)
		if err != nil {
			debug("Error trying to unmarshal %s", text)
			errs = append(errs, err.Error())
			continue
		}

		target := field
		if f.target != nil {
			target = *f.target
		}

		if target != "" {
			_, err = event.PutValue(target, output)
		} else {
			switch t := output.(type) {
			case map[string]interface{}:
				jsontransform.WriteJSONKeys(event, t, f.overwriteKeys)
			default:
				errs = append(errs, "failed to add target to root")
			}
		}

		if err != nil {
			debug("Error trying to Put value %v for field : %s", output, field)
			errs = append(errs, err.Error())
			continue
		}
	}

	if len(errs) > 0 {
		return event, fmt.Errorf(strings.Join(errs, ", "))
	}
	return event, nil
}

func unmarshal(maxDepth int, text string, fields *interface{}, processArray bool) error {
	if err := decodeJSON(text, fields); err != nil {
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
		err := unmarshal(maxDepth, str, &tmp, processArray)
		if err != nil {
			return v, err == errProcessingSkipped
		}

		return tmp, true
	}

	// try to deep unmarshal fields
	switch O := interface{}(*fields).(type) {
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

func decodeJSON(text string, to *interface{}) error {
	dec := json.NewDecoder(strings.NewReader(text))
	dec.UseNumber()
	err := dec.Decode(to)

	if err != nil {
		return err
	}

	if dec.More() {
		return errors.New("multiple json elements found")
	}

	if _, err := dec.Token(); err != nil && err != io.EOF {
		return err
	}

	switch O := interface{}(*to).(type) {
	case map[string]interface{}:
		jsontransform.TransformNumbers(O)
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
