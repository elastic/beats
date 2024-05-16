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

package pulsar

import (
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/outputs/codec"
	"github.com/elastic/elastic-agent-libs/config"
)

// init registers the pulsar output.
func init() {
	outputs.RegisterType("pulsar", newPulsar)
}

// newPulsar creates a new pulsar output.
func newPulsar(
	_ outputs.IndexManager,
	info beat.Info,
	stats outputs.Observer,
	cfg *config.C) (outputs.Group, error) {
	config0, err := readConfig(cfg)
	if err != nil {
		return outputs.Fail(err)
	}
	codec0, err := codec.CreateEncoder(info, config0.Codec)
	if err != nil {
		return outputs.Fail(err)
	}

	return outputs.Success(config0.Queue, config0.BulkMaxSize, config0.MaxRetries, nil, &client{
		config:   config0,
		codec:    codec0,
		observer: stats,
		index:    info.IndexPrefix,
	})
}
