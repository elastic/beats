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

package decode_duration

import (
	"fmt"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/processors/checks"
	jsprocessor "github.com/elastic/beats/v7/libbeat/processors/script/javascript/module/processor"
	"github.com/elastic/elastic-agent-libs/config"
)

func init() {
	processors.RegisterPlugin("decode_duration",
		checks.ConfigChecked(NewDecodeDuration,
			checks.RequireFields("field", "format")))
	jsprocessor.RegisterPlugin("DecodeDuration", NewDecodeDuration)
}

type decodeDurationConfig struct {
	Field  string `config:"field"`
	Format string `config:"format"`
}

type decodeDuration struct {
	config decodeDurationConfig
}

func (u decodeDuration) Run(event *beat.Event) (*beat.Event, error) {
	fields := event.Fields
	fieldName := u.config.Field
	x, err := fields.GetValue(fieldName)
	if err != nil {
		return event, fmt.Errorf("field '%s' not present on event", fieldName)
	}
	durationString, ok := x.(string)
	if !ok {
		return event, fmt.Errorf("field '%s' is not a string, value: '%#v'", fieldName, x)
	}
	d, err := time.ParseDuration(durationString)
	if err != nil {
		return event, fmt.Errorf("couldn't parse field '%s' as duration: %w", fieldName, err)
	}
	switch u.config.Format {
	case "milliseconds":
		// keep the result is type float64
		x = float64(d.Milliseconds())
	case "seconds":
		x = d.Seconds()
	case "minutes":
		x = d.Minutes()
	case "hours":
		x = d.Hours()
	default:
		x = float64(d.Milliseconds())
	}
	if _, err = fields.Put(fieldName, x); err != nil {
		return event, fmt.Errorf("put field '%s' back to event failed: %w", fieldName, err)
	}
	return event, nil
}

func (u decodeDuration) String() string {
	return "decode_duration"
}

func NewDecodeDuration(c *config.C) (beat.Processor, error) {
	fc := decodeDurationConfig{}
	err := c.Unpack(&fc)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack decode duration config: %w", err)
	}

	return &decodeDuration{
		config: fc,
	}, nil
}
