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

package dissect

import (
	"errors"
	"fmt"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/processors"
	jsprocessor "github.com/elastic/beats/v7/libbeat/processors/script/javascript/module/processor/registry"
	cfg "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const flagParsingError = "dissect_parsing_error"

type processor struct {
	config config
}

func init() {
	processors.RegisterPlugin("dissect", NewProcessor)
	jsprocessor.RegisterPlugin("Dissect", NewProcessor)
}

// NewProcessor constructs a new dissect processor.
func NewProcessor(c *cfg.C, log *logp.Logger) (beat.Processor, error) {
	config := defaultConfig
	err := c.Unpack(&config)
	if err != nil {
		return nil, err
	}
	if config.TrimValues != trimModeNone {
		config.Tokenizer.trimmer, err = newTrimmer(config.TrimChars,
			config.TrimValues&trimModeLeft != 0,
			config.TrimValues&trimModeRight != 0)
		if err != nil {
			return nil, err
		}
	}
	p := &processor{config: config}

	return p, nil
}

// Run takes the event and will apply the tokenizer on the configured field.
func (p *processor) Run(event *beat.Event) (*beat.Event, error) {
	var (
		m   Map
		mc  MapConverted
		v   interface{}
		err error
	)

	v, err = event.GetValue(p.config.Field)
	if err != nil {
		return event, err
	}

	s, ok := v.(string)
	if !ok {
		return event, fmt.Errorf("field is not a string, value: `%v`, field: `%s`", v, p.config.Field)
	}

	convertDataType := false
	for _, f := range p.config.Tokenizer.parser.fields {
		if f.DataType() != "" {
			convertDataType = true
		}
	}

	if convertDataType {
		mc, err = p.config.Tokenizer.DissectConvert(s)
	} else {
		m, err = p.config.Tokenizer.Dissect(s)
	}
	if err != nil {
		if err := mapstr.AddTagsWithKey(
			event.Fields,
			beat.FlagField,
			[]string{flagParsingError},
		); err != nil {
			return event, fmt.Errorf("cannot add new flag the event: %w", err)
		}
		if p.config.IgnoreFailure {
			return event, nil
		}
		return event, err
	}

	if convertDataType {
		event, err = p.mapper(event, mapInterfaceToMapStr(mc))
	} else {
		event, err = p.mapper(event, mapToMapStr(m))
	}

	return event, err
}

func (p *processor) mapper(event *beat.Event, m mapstr.M) (*beat.Event, error) {
	prefix := ""
	if p.config.TargetPrefix != "" {
		prefix = p.config.TargetPrefix + "."
	}

	// Check all keys first so we never need to clone the event for rollback.
	if !p.config.OverwriteKeys {
		for k := range m {
			prefixKey := prefix + k
			found, err := event.HasKey(prefixKey)
			if found {
				return event, fmt.Errorf("cannot override existing key with `%s`", prefixKey)
			}
			if err != nil && !errors.Is(err, mapstr.ErrKeyNotFound) {
				// Path traverses a non-map value — treat as conflict.
				return event, fmt.Errorf("cannot override existing key with `%s`: %w", prefixKey, err)
			}
		}
	}

	for k, v := range m {
		_, _ = event.PutValue(prefix+k, v)
	}

	return event, nil
}

func (p *processor) String() string {
	return "dissect=" + p.config.Tokenizer.Raw() +
		",field=" + p.config.Field +
		",target_prefix=" + p.config.TargetPrefix
}

func mapToMapStr(m Map) mapstr.M {
	newMap := make(mapstr.M, len(m))
	for k, v := range m {
		newMap[k] = v
	}
	return newMap
}

func mapInterfaceToMapStr(m MapConverted) mapstr.M {
	newMap := make(mapstr.M, len(m))
	for k, v := range m {
		newMap[k] = v
	}
	return newMap
}
