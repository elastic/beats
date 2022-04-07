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

package udp

import (
	"time"

	"github.com/dustin/go-humanize"

	"github.com/elastic/beats/v8/filebeat/harvester"
	"github.com/elastic/beats/v8/filebeat/inputsource/udp"
)

var defaultConfig = config{
	ForwarderConfig: harvester.ForwarderConfig{
		Type: "udp",
	},
	Config: udp.Config{
		MaxMessageSize: 10 * humanize.KiByte,
		// TODO: What should be default port?
		Host: "localhost:8080",
		// TODO: What should be the default timeout?
		Timeout: time.Minute * 5,
	},
}

type config struct {
	udp.Config                `config:",inline"`
	harvester.ForwarderConfig `config:",inline"`
}
