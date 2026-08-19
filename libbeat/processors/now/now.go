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

package now

import (
	"fmt"
	"time"

	"github.com/elastic/beats/v9/libbeat/beat"
	"github.com/elastic/beats/v9/libbeat/processors"
	"github.com/elastic/beats/v9/libbeat/processors/checks"
	jsprocessor "github.com/elastic/beats/v9/libbeat/processors/script/javascript/module/processor/registry"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

var currentTime = time.Now

type now struct {
	config nowConfig
	log    *logp.Logger
}

type nowConfig struct {
	Field string `config:"field" validate:"required"`
}

func init() {
	processors.RegisterPlugin("now",
		checks.ConfigChecked(New,
			checks.RequireFields("field"),
			checks.AllowedFields("field")))
	jsprocessor.RegisterPlugin("Now", New)
}

func New(c *config.C, log *logp.Logger) (beat.Processor, error) {
	nowConfig := nowConfig{}

	if err := c.Unpack(&nowConfig); err != nil {
		return nil, fmt.Errorf("failed to unpack the configuration of now processor: %w", err)
	}

	return &now{
		config: nowConfig,
		log:    log.Named("now"),
	}, nil

}

func (n *now) Run(event *beat.Event) (*beat.Event, error) {
	_, err := event.PutValue(n.config.Field, currentTime())
	return event, err
}

func (n *now) String() string {
	return "now=" + fmt.Sprintf("%+v", n.config.Field)
}
