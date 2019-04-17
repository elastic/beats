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

package kafka

import (
	"fmt"
	"time"

	"github.com/elastic/beats/libbeat/common/transport/tlscommon"
)

type metricsetConfig struct {
	Retries  int               `config:"retries" validate:"min=0"`
	Backoff  time.Duration     `config:"backoff" validate:"min=0"`
	TLS      *tlscommon.Config `config:"ssl"`
	Username string            `config:"username"`
	Password string            `config:"password"`
	ClientID string            `config:"client_id"`
}

var defaultConfig = metricsetConfig{
	Retries:  3,
	Backoff:  250 * time.Millisecond,
	TLS:      nil,
	Username: "",
	Password: "",
	ClientID: "metricbeat",
}

func (c *metricsetConfig) Validate() error {
	if c.Username != "" && c.Password == "" {
		return fmt.Errorf("password must be set when username is configured")
	}

	return nil
}
