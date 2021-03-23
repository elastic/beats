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

package decode_xml_wineventlog

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/jsontransform"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/processors/checks"
	jsprocessor "github.com/elastic/beats/v7/libbeat/processors/script/javascript/module/processor"
	"github.com/elastic/beats/v7/winlogbeat/sys/winevent"
)

type processor struct {
	config
	log *logp.Logger
}

var (
	errFieldIsNotString = errors.New("field value is not a string")
)

const (
	procName = "decode_xml_wineventlog"
	logName  = "processor." + procName
)

func init() {
	processors.RegisterPlugin(procName,
		checks.ConfigChecked(New,
			checks.RequireFields("field"),
			checks.AllowedFields(
				"field", "overwrite_keys",
				"target_field", "ignore_missing",
				"ignore_failure",
			)))
	jsprocessor.RegisterPlugin(procName, New)
}

// New constructs a new decode_xml processor.
func New(c *common.Config) (processors.Processor, error) {
	config := defaultConfig()

	if err := c.Unpack(&config); err != nil {
		return nil, fmt.Errorf("fail to unpack the "+procName+" processor configuration: %s", err)
	}

	return newProcessor(config)
}

func newProcessor(config config) (processors.Processor, error) {
	// Default target to overwriting field.
	if config.Target == nil {
		config.Target = &config.Field
	}

	return &processor{
		config: config,
		log:    logp.NewLogger(logName),
	}, nil
}

func (p *processor) Run(event *beat.Event) (*beat.Event, error) {
	if err := p.run(event); err != nil && !p.IgnoreFailure {
		err = fmt.Errorf("failed in decode_xml_wineventlog on the %q field: %w", p.Field, err)
		_, _ = event.PutValue("error.message", err.Error())
		return event, err
	}
	return event, nil
}

func (p *processor) run(event *beat.Event) error {
	data, err := event.GetValue(p.Field)
	if err != nil {
		if p.IgnoreMissing && err == common.ErrKeyNotFound {
			return nil
		}
		return err
	}

	text, ok := data.(string)
	if !ok {
		return errFieldIsNotString
	}

	winevt, err := p.decode(text)
	if err != nil {
		return err
	}

	if *p.Target != "" {
		if _, err = event.PutValue(*p.Target, winevt); err != nil {
			return fmt.Errorf("failed to put value %v into field %q: %w", winevt, *p.Target, err)
		}
	} else {
		jsontransform.WriteJSONKeys(event, winevt, false, p.OverwriteKeys, !p.IgnoreFailure)
	}

	return nil
}

func (p *processor) decode(data string) (common.MapStr, error) {
	evt, err := winevent.UnmarshalXML([]byte(data))
	if err != nil {
		return nil, err
	}
	return evt.Fields(), nil
}

func (p *processor) String() string {
	json, _ := json.Marshal(p.config)
	return procName + "=" + string(json)
}
