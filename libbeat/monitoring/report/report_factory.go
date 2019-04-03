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

package report

import (
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/cfgfile"
	"github.com/elastic/beats/libbeat/common"
)

// Factory is factory producing reporters
type Factory struct {
	beat     beat.Info
	outputs  common.ConfigNamespace
	settings Settings
}

// NewFactory returns a factory for creating instances of
func NewFactory(beat beat.Info, outputs common.ConfigNamespace, settings Settings) *Factory {
	return &Factory{
		beat:     beat,
		outputs:  outputs,
		settings: settings,
	}
}

// Create creates a reporter based on a config
func (f *Factory) Create(p beat.Pipeline, c *common.Config, meta *common.MapStrPointer) (cfgfile.Runner, error) {
	reporter, err := New(f.beat, f.settings, c, f.outputs)
	return reporter, err
}

// CheckConfig checks if a config is valid or not
func (f *Factory) CheckConfig(config *common.Config) error {
	// TODO: add code here once we know that spinning up a filebeat input to check for errors doesn't cause memory leaks.
	return nil
}
