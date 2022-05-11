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
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/libbeat/common/jsontransform"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/processors/checks"
	jsprocessor "github.com/elastic/beats/v7/libbeat/processors/script/javascript/module/processor"
	"github.com/elastic/beats/v7/winlogbeat/sys/winevent"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

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
			checks.RequireFields("field", "target_field"),
			checks.AllowedFields(
				"field", "target_field",
				"overwrite_keys", "map_ecs_fields",
				"ignore_missing", "ignore_failure",
				"when",
			)))
	jsprocessor.RegisterPlugin("DecodeXMLWineventlog", New)
}

type processor struct {
	config

	decoder decoder
	log     *logp.Logger
}

type decoder interface {
	decode(data []byte) (win, ecs mapstr.M, err error)
}

// New constructs a new decode_xml processor.
func New(c *conf.C) (processors.Processor, error) {
	config := defaultConfig()

	if err := c.Unpack(&config); err != nil {
		return nil, fmt.Errorf("fail to unpack the "+procName+" processor configuration: %s", err)
	}

	return newProcessor(config)
}

func newProcessor(config config) (processors.Processor, error) {
	cfgwarn.Experimental("The " + procName + " processor is experimental.")

	return &processor{
		config:  config,
		decoder: newDecoder(),
		log:     logp.NewLogger(logName),
	}, nil
}

func (p *processor) Run(event *beat.Event) (*beat.Event, error) {
	if err := p.run(event); err != nil && !p.IgnoreFailure {
		err = fmt.Errorf("failed in decode_xml_wineventlog on the %q field: %w", p.Field, err)
		event.PutValue("error.message", err.Error())
		return event, err
	}
	return event, nil
}

func (p *processor) run(event *beat.Event) error {
	data, err := event.GetValue(p.Field)
	if err != nil {
		if p.IgnoreMissing && err == mapstr.ErrKeyNotFound {
			return nil
		}
		return err
	}

	text, ok := data.(string)
	if !ok {
		return errFieldIsNotString
	}

	win, ecs, err := p.decoder.decode([]byte(text))
	if err != nil {
		return fmt.Errorf("error decoding XML field: %w", err)
	}

	if p.Target != "" {
		if _, err = event.PutValue(p.Target, win); err != nil {
			return fmt.Errorf("failed to put value %v into field %q: %w", win, p.Target, err)
		}
	} else {
		jsontransform.WriteJSONKeys(event, win, false, p.OverwriteKeys, !p.IgnoreFailure)
	}

	if p.MapECSFields {
		jsontransform.WriteJSONKeys(event, ecs, false, p.OverwriteKeys, !p.IgnoreFailure)
	}

	return nil
}

func (p *processor) String() string {
	json, _ := json.Marshal(p.config)
	return procName + "=" + string(json)
}

func fields(evt winevent.Event) (mapstr.M, mapstr.M) {
	win := evt.Fields()

	ecs := mapstr.M{}

	eventCode, _ := win.GetValue("event_id")
	ecs.Put("event.code", eventCode)
	ecs.Put("event.kind", "event")
	ecs.Put("event.provider", evt.Provider.Name)
	winevent.AddOptional(ecs, "event.action", evt.Task)
	winevent.AddOptional(ecs, "host.name", evt.Computer)
	winevent.AddOptional(ecs, "event.outcome", getValue(win, "outcome"))
	winevent.AddOptional(ecs, "log.level", getValue(win, "level"))
	winevent.AddOptional(ecs, "message", getValue(win, "message"))
	winevent.AddOptional(ecs, "error.code", getValue(win, "error.code"))
	winevent.AddOptional(ecs, "error.message", getValue(win, "error.message"))

	return win, ecs
}

func getValue(m mapstr.M, key string) interface{} {
	v, _ := m.GetValue(key)
	return v
}
