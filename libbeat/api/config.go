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

package api

import "os"

// Config is the configuration for the API endpoint.
type Config struct {
	Enabled            bool   `config:"enabled"`
	Host               string `config:"host"`
	Port               int    `config:"port"`
	User               string `config:"named_pipe.user"`
	SecurityDescriptor string `config:"named_pipe.security_descriptor"`
}

var (
	// DefaultConfig is the default configuration used by the API endpoint.
	DefaultConfig = Config{
		Enabled: false,
		Host:    "localhost",
		Port:    5066,
	}
)

// File mode for the socket file, owner of the process can do everything, member of the group can read.
const socketFileMode = os.FileMode(0740)
