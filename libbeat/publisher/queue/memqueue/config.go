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

	"github.com/elastic/beats/v7/libbeat/common/cfgtype"
	c "github.com/elastic/elastic-agent-libs/config"
)

type config struct {
	Events *int              `config:"events" validate:"min=32"`
	Bytes  *cfgtype.ByteSize `config:"bytes"`

	// This field is named MaxGetEvents because its logical effect is to give
	// a maximum on the number of events a Get request can return, but the
	// user-exposed name is "flush.min_events" for backwards compatibility,
	// since it used to control buffer size in the internal buffer chain.
	// Ignored if a byte limit is set in the queue or the get request.
	MaxGetEvents int           `config:"flush.min_events" validate:"min=0"`
	FlushTimeout time.Duration `config:"flush.timeout"`
}

const minQueueBytes = 32768
const minQueueEvents = 32

func (c *config) Validate() error {
	if c.Bytes != nil && *c.Bytes < minQueueBytes {
		return errors.New(fmt.Sprintf("queue byte size must be at least %v", minQueueBytes))
	}
	if c.Events != nil && *c.Events < minQueueEvents {
		return errors.New(fmt.Sprintf("queue event size must be at least %v", minQueueEvents))
	}
	if c.Events == nil && c.Bytes == nil {
		return errors.New("queue must have an event limit or a byte limit")
	}
	if c.Events != nil && c.MaxGetEvents > *c.Events {
		return errors.New("flush.min_events must be less than events")
	}
	return nil
}

var defaultConfig = config{
	MaxGetEvents: 1600,
	FlushTimeout: 10 * time.Second,
}

// SettingsForUserConfig unpacks a ucfg config from a Beats queue
// configuration and returns the equivalent memqueue.Settings object.
func SettingsForUserConfig(cfg *c.C) (Settings, error) {
	var config config
	if cfg != nil {
		if err := cfg.Unpack(&config); err != nil {
			return Settings{}, fmt.Errorf("couldn't unpack memory queue config: %w", err)
		}
	}
	result := Settings{
		MaxGetRequest: config.MaxGetEvents,
		FlushTimeout:  config.FlushTimeout,
	}
	if config.Events != nil {
		result.Events = *config.Events
	}
	if config.Bytes != nil {
		result.Bytes = int(*config.Bytes)
	}
	// If no size constraint was given, fall back on the default event cap
	if config.Events == nil && config.Bytes == nil {
		result.Events = 3200
	}
	return result, nil
}
