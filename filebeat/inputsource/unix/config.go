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

package unix

import (
	"fmt"
	"time"

	"github.com/elastic/beats/v7/libbeat/common/cfgtype"
)

// Name is the human readable name and identifier.
const Name = "unix"

// Config exposes the unix configuration.
type Config struct {
	Path           string           `config:"path"`
	Group          *string          `config:"group"`
	Mode           *string          `config:"mode"`
	Timeout        time.Duration    `config:"timeout" validate:"nonzero,positive"`
	MaxMessageSize cfgtype.ByteSize `config:"max_message_size" validate:"nonzero,positive"`
	MaxConnections int              `config:"max_connections"`
}

// Validate validates the Config option for the unix input.
func (c *Config) Validate() error {
	if len(c.Path) == 0 {
		return fmt.Errorf("need to specify the path to the unix socket")
	}
	return nil
}
