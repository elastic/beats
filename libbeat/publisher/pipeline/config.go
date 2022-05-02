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

package pipeline

import (
	"errors"
	"fmt"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// Config object for loading a pipeline instance via Load.
type Config struct {
	// Event processing configurations
	mapstr.EventMetadata `config:",inline"`      // Fields and tags to add to each event.
	Processors           processors.PluginConfig `config:"processors"`

	// Event queue
	Queue config.Namespace `config:"queue"`
}

// validateClientConfig checks a ClientConfig can be used with (*Pipeline).ConnectWith.
func validateClientConfig(c *beat.ClientConfig) error {
	withDrop := false

	switch m := c.PublishMode; m {
	case beat.DefaultGuarantees, beat.GuaranteedSend, beat.OutputChooses:
	case beat.DropIfFull:
		withDrop = true
	default:
		return fmt.Errorf("unknown publish mode %v", m)
	}

	// ACK handlers can not be registered DropIfFull is set, as dropping events
	// due to full broker can not be accounted for in the clients acker.
	if c.ACKHandler != nil && withDrop {
		return errors.New("ACK handlers with DropIfFull mode not supported")
	}

	return nil
}
