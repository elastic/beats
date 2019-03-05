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

package channel

import (
	"github.com/elastic/beats/filebeat/util"
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

// Factory is used to create a new Outlet instance
type Factory func(beat.Pipeline, *common.Config, *common.MapStrPointer) (Outleter, error)

// Connector creates an Outlet connecting the event publishing with some internal pipeline.
type Connector func(*common.Config, *common.MapStrPointer) (Outleter, error)

// Outleter is the outlet for an input
type Outleter interface {
	Close() error
	OnEvent(data *util.Data) bool
}
