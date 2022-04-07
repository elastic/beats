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

package add_id

import (
	"fmt"

	"github.com/elastic/beats/v8/libbeat/processors/add_id/generator"

	"github.com/elastic/beats/v8/libbeat/beat"
	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/processors"
	jsprocessor "github.com/elastic/beats/v8/libbeat/processors/script/javascript/module/processor"
)

func init() {
	processors.RegisterPlugin("add_id", New)
	jsprocessor.RegisterPlugin("AddID", New)
}

const processorName = "add_id"

type addID struct {
	config config
	gen    generator.IDGenerator
}

// New constructs a new Add ID processor.
func New(cfg *common.Config) (processors.Processor, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, makeErrConfigUnpack(err)
	}

	gen, err := generator.Factory(config.Type)
	if err != nil {
		return nil, makeErrComputeID(err)
	}

	p := &addID{
		config,
		gen,
	}

	return p, nil
}

// Run enriches the given event with an ID
func (p *addID) Run(event *beat.Event) (*beat.Event, error) {
	id := p.gen.NextID()

	if _, err := event.PutValue(p.config.TargetField, id); err != nil {
		return nil, makeErrComputeID(err)
	}

	return event, nil
}

func (p *addID) String() string {
	return fmt.Sprintf("%v=[target_field=[%v]]", processorName, p.config.TargetField)
}
