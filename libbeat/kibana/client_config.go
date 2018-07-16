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

package kibana

import (
	"time"

	"github.com/elastic/beats/libbeat/common/transport/tlscommon"
)

// ClientConfig to connect to Kibana
type ClientConfig struct {
	Protocol string            `config:"protocol"`
	Host     string            `config:"host"`
	Path     string            `config:"path"`
	Username string            `config:"username"`
	Password string            `config:"password"`
	TLS      *tlscommon.Config `config:"ssl"`
	Timeout  time.Duration     `config:"timeout"`
}

var (
	defaultClientConfig = ClientConfig{
		Protocol: "http",
		Host:     "localhost:5601",
		Path:     "",
		Username: "",
		Password: "",
		Timeout:  90 * time.Second,
		TLS:      nil,
	}
)
