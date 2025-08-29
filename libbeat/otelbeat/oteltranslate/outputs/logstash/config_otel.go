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

package logstash

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/logstash"
	"github.com/elastic/elastic-agent-libs/config"
)

// ToMap converts Logstash output config to a map[string]any
// raises error if config is invalid
func ToMap(outputConfig *config.C) (map[string]any, error) {
	lsHostWorker := struct {
		outputs.HostWorkerCfg `config:",inline"`
		logstash.Config       `config:",inline"`
	}{
		Config: logstash.DefaultConfig(),
	}

	// unpack and validate LS config
	if err := outputConfig.Unpack(&lsHostWorker); err != nil {
		return nil, fmt.Errorf("failed unpacking logstash config: %w", err)
	}

	lsConfig := config.MustNewConfigFrom(lsHostWorker)
	var lsConfigMap map[string]any
	if err := lsConfig.Unpack(&lsConfigMap); err != nil {
		return nil, err
	}

	return lsConfigMap, nil
}
