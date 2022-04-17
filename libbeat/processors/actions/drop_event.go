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

package actions

import (
	"github.com/menderesk/beats/v7/libbeat/beat"
	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/processors"
	"github.com/menderesk/beats/v7/libbeat/processors/checks"
)

type dropEvent struct{}

func init() {
	processors.RegisterPlugin("drop_event",
		checks.ConfigChecked(newDropEvent, checks.AllowedFields("when")))
}

var dropEventsSingleton = (*dropEvent)(nil)

func newDropEvent(c *common.Config) (processors.Processor, error) {
	return dropEventsSingleton, nil
}

func (*dropEvent) Run(_ *beat.Event) (*beat.Event, error) {
	// return event=nil to delete the entire event
	return nil, nil
}

func (*dropEvent) String() string { return "drop_event" }
