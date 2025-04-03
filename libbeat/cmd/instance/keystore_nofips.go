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

//go:build !requirefips

package instance

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/keystore"
	"github.com/elastic/elastic-agent-libs/paths"
)

// LoadKeystore returns the appropriate keystore based on the configuration.
func LoadKeystore(cfg *config.C, name string) (keystore.Keystore, error) {
	keystoreCfg, _ := cfg.Child("keystore", -1)
	defaultPathConfig := paths.Resolve(paths.Data, fmt.Sprintf("%s.keystore", name))
	return keystore.Factory(keystoreCfg, defaultPathConfig, common.IsStrictPerms())
}
