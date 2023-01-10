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

package inputs

import (
	"github.com/elastic/beats/v7/filebeat/input/journald"
	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	cursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
)

// inputs that are only supported on linux

type osComponents interface {
	cursor.StateStore
}

func osInputs(info beat.Info, log *logp.Logger, components osComponents) []v2.Plugin {
	var plugins []v2.Plugin

	zeroPlugin := v2.Plugin{}
	if journald := journald.Plugin(log, components); journald != zeroPlugin {
		plugins = append(plugins, journald)
	}

	return plugins
}
