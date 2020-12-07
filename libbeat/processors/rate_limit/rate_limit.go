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

package rate_limit

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/processors/rate_limit/algorithm"
	jsprocessor "github.com/elastic/beats/v7/libbeat/processors/script/javascript/module/processor"
)

func init() {
	processors.RegisterPlugin("rate_limit", New)
	jsprocessor.RegisterPlugin("Fingerprint", New)
}

const processorName = "rate_limit"

type rateLimit struct {
	config    Config
	algorithm algorithm.Algorithm
}

// New constructs a new rate limit processor.
func New(cfg *common.Config) (processors.Processor, error) {
	config, err := defaultConfig()
	if err != nil {
		return nil, errors.Wrap(err, "could not create default configuration")
	}

	if err := cfg.Unpack(&config); err != nil {
		// TODO: make custom error: errConfigUnpack?
		return nil, errors.Wrap(err, "could not unpack processor configuration")
	}

	algoCtor, err := algorithm.Factory(config.Algorithm.Name())
	if err != nil {
		return nil, errors.Wrap(err, "could not instantiate rate limiting algorithm")
	}

	algo := algoCtor(algorithm.Config{
		Limit:  config.Limit,
		Config: *config.Algorithm.Config(),
	})

	// TODO: flesh out fields
	p := &rateLimit{
		config:    *config,
		algorithm: algo,
	}

	return p, nil
}

// Run applies the configured rate limit to the given event. If the event is within the
// configured rate limit, it is returned as-is. If not, nil is returned.
func (p *rateLimit) Run(event *beat.Event) (*beat.Event, error) {
	key := "" // TODO: construct key from event fields + config

	if p.algorithm.IsAllowed(key) {
		return event, nil
	}

	// TODO: log that event is being dropped
	return nil, nil
}

func (p *rateLimit) String() string {
	return fmt.Sprintf(
		"%v=[limit=[%v],fields=[%v],algorithm=[%v]]",
		processorName, p.config.Limit, p.config.Fields, p.config.Algorithm.Name(),
	)
}
