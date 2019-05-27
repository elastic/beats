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

package docker

var defaultConfig = config{
	Partial: true,
	Containers: containers{
		IDs:    []string{},
		Path:   "/var/lib/docker/containers",
		Stream: "all",
	},
}

type config struct {
	// List of containers' log files to tail
	Containers containers `config:"containers"`

	// Partial configures the input to join partial lines
	Partial bool `config:"combine_partials"`

	// Enable CRI flags parsing (to be switched to default in 7.0)
	CRIFlags bool `config:"cri.parse_flags"`

	// Fore CRI format (don't perform autodetection)
	CRIForce bool `config:"cri.force"`
}

type containers struct {
	IDs  []string `config:"ids"`
	Path string   `config:"path"`

	// Stream can be all, stdout or stderr
	Stream string `config:"stream"`
}
