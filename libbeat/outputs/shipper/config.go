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

package shipper

import (
	"time"

	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
)

type backoffConfig struct {
	Init time.Duration `config:"init" validate:"nonzero"`
	Max  time.Duration `config:"max" validate:"nonzero"`
}

type Config struct {
	// Server address in the format of host:port, e.g. `localhost:50051`
	Server string `config:"server"`
	// TLS/SSL configurationf or secure connection
	TLS *tlscommon.Config `config:"ssl"`
	// Timeout of a single batch publishing request
	Timeout time.Duration `config:"timeout" validate:"min=1"`
	// MaxRetries is how many times the same batch is attempted to be sent
	MaxRetries int `config:"max_retries" validate:"min=-1,nonzero"`
	// BulkMaxSize max amount of events in a single batch
	BulkMaxSize int `config:"bulk_max_size"`
	// AckPollingInterval is a minimal interval for getting persisted index updates from the shipper server.
	// Batches of events are acknowledged asynchronously in the background.
	// If after the `AckPollingInterval` duration the persisted index value changed
	// all batches pending acknowledgment will be checked against the new value
	// and acknowledged if `persisted_index` >= `accepted_index`.
	AckPollingInterval time.Duration `config:"ack_polling_interval" validate:"min=5ms"`
	// Backoff strategy for the shipper output
	Backoff backoffConfig `config:"backoff"`
}

func defaultConfig() Config {
	return Config{
		TLS:                nil,
		Timeout:            30 * time.Second,
		MaxRetries:         3,
		BulkMaxSize:        50,
		AckPollingInterval: 5 * time.Millisecond,
		Backoff: backoffConfig{
			Init: 1 * time.Second,
			Max:  60 * time.Second,
		},
	}
}
