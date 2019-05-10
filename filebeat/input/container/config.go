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

package container

var defaultConfig = config{
	Stream: "all",
	Format: "auto",
}

type config struct {
	// Stream can be all, stdout or stderr
	Stream string `config:"stream"`

	// Fore CRI format (don't perform autodetection)
	// TODO remove in favor of format, below
	CRIForce bool `config:"cri.force"`

	// TODO Format can be auto, cri, json-file
	Format string `config:"format"`
}
