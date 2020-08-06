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

package dot_expand

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/processors"
	jsprocessor "github.com/elastic/beats/v7/libbeat/processors/script/javascript/module/processor"
)

type config struct {
	FailOnError bool `config:"fail_on_error"`
}

var (
	defaultConfig = config{
		FailOnError: true,
	}
)

func init() {
	processors.RegisterPlugin("dot_expand", New)
	jsprocessor.RegisterPlugin("DotExpand", New)
}

type dotExpandProcessor struct {
	config
}

// Unpack unpacks the processor's configuration.
func (f *dotExpandProcessor) Unpack(from *common.Config) error {
	tmp := defaultConfig
	err := from.Unpack(&tmp)
	if err != nil {
		return fmt.Errorf("failed to unpack the dot_expand configuration: %s", err)
	}
	f.config = tmp
	return nil
}

// New builds a new dot_expand processor.
func New(c *common.Config) (processors.Processor, error) {
	p := &dotExpandProcessor{}
	err := c.Unpack(p)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (f *dotExpandProcessor) Run(event *beat.Event) (*beat.Event, error) {
	x, err := event.Fields.Expand()
	if err != nil {
		if f.FailOnError {
			return event, err
		} else {
			return event, nil
		}
	}
	event.Fields = x
	return event, nil
}

func (f *dotExpandProcessor) String() (r string) {
	return "dot_expand"
}
