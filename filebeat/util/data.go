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

package util

import (
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"

	"github.com/elastic/beats/filebeat/input/file"
)

type Data struct {
	Event beat.Event
	state file.State
}

func NewData() *Data {
	return &Data{}
}

// SetState sets the state
func (d *Data) SetState(state file.State) {
	d.state = state
}

// GetState returns the current state
func (d *Data) GetState() file.State {
	return d.state
}

// HasState returns true if the data object contains state data
func (d *Data) HasState() bool {
	return !d.state.IsEmpty()
}

// GetEvent returns the event in the data object
// In case meta data contains module and fileset data, the event is enriched with it
func (d *Data) GetEvent() beat.Event {
	return d.Event
}

// GetMetadata creates a common.MapStr containing the metadata to
// be associated with the event.
func (d *Data) GetMetadata() common.MapStr {
	return d.Event.Meta
}

// HasEvent returns true if the data object contains event data
func (d *Data) HasEvent() bool {
	return d.Event.Fields != nil
}
