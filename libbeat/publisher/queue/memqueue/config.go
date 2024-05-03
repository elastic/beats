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

package memqueue

import (
	"errors"
	"fmt"
	"time"

	c "github.com/elastic/elastic-agent-libs/config"
)

type config struct {
	Events int `config:"events" validate:"min=32"`
	Bytes  int `config:"bytes" validate:"min=32768"`

	// This field is named MaxGetEvents because its logical effect is to give
	// a maximum on the number of events a Get request can return, but the
	// user-exposed name is "flush.min_events" for backwards compatibility,
	// since it used to control buffer size in the internal buffer chain.
	// Ignored if a byte limit is set in the queue or the get request.
	MaxGetEvents int           `config:"flush.min_events" validate:"min=0"`
	FlushTimeout time.Duration `config:"flush.timeout"`
}

var defaultConfig = config{
	Events:       3200,
	MaxGetEvents: 1600,
	FlushTimeout: 10 * time.Second,
}

func (c *config) Validate() error {
	if c.MaxGetEvents > c.Events {
		return errors.New("flush.min_events must be less events")
	}
	return nil
}

// SettingsForUserConfig unpacks a ucfg config from a Beats queue
// configuration and returns the equivalent memqueue.Settings object.
func SettingsForUserConfig(cfg *c.C) (Settings, error) {
	config := defaultConfig
	if cfg != nil {
		if err := cfg.Unpack(&config); err != nil {
			return Settings{}, fmt.Errorf("couldn't unpack memory queue config: %w", err)
		}
	}
	//nolint:gosimple // Actually want this conversion to be explicit since the types aren't definitionally equal.
	return Settings{
		Events:        config.Events,
		MaxGetRequest: config.MaxGetEvents,
		FlushTimeout:  config.FlushTimeout,
	}, nil
}
