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

package harvester

import (
	"errors"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
)

// Outlet interface is used for forwarding events
type Outlet interface {
	OnEvent(data beat.Event) bool
}

// Forwarder contains shared options between all harvesters needed to forward events
type Forwarder struct {
	Outlet Outlet
}

// ForwarderConfig contains all config options shared by all harvesters
type ForwarderConfig struct {
	Type string `config:"type"`
}

// NewForwarder creates a new forwarder instances and initialises processors if configured
func NewForwarder(outlet Outlet) *Forwarder {
	return &Forwarder{Outlet: outlet}
}

// Send updates the input state and sends the event to the spooler
// All state updates done by the input itself are synchronous to make sure no states are overwritten
func (f *Forwarder) Send(event beat.Event) error {
	ok := f.Outlet.OnEvent(event)
	if !ok {
		logp.Info("Input outlet closed")
		return errors.New("input outlet closed")
	}

	return nil
}
