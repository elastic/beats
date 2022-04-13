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

	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"

	"google.golang.org/grpc/backoff"
)

type Backoff struct {
	// BaseDelay is the amount of time to backoff after the first failure.
	BaseDelay time.Duration `config:"base_delay"`
	// Multiplier is the factor with which to multiply backoffs after a
	// failed retry. Should ideally be greater than 1.
	Multiplier float64 `config:"multiplier"`
	// Jitter is the factor with which backoffs are randomized.
	Jitter float64 `config:"jitter"`
	// MaxDelay is the upper bound of backoff delay.
	MaxDelay time.Duration `config:"max_delay"`
}

func (b Backoff) ToGRPCBackOff() backoff.Config {
	return backoff.Config{
		BaseDelay:  b.BaseDelay,
		Multiplier: b.Multiplier,
		Jitter:     b.Jitter,
		MaxDelay:   b.MaxDelay,
	}
}
func FromGRPCBackOff(b backoff.Config) Backoff {
	return Backoff{
		BaseDelay:  b.BaseDelay,
		Multiplier: b.Multiplier,
		Jitter:     b.Jitter,
		MaxDelay:   b.MaxDelay,
	}
}

type Config struct {
	// Server address in the format of host:port, e.g. `localhost:50051`
	Server string `config:"server"`
	// TLS/SSL configurationf or secure connection
	TLS *tlscommon.Config `config:"ssl"`
	// Timeout of a single batch publishing request
	Timeout time.Duration `config:"timeout"             validate:"min=1"`
	// MaxRetries is how many times the same batch is attempted to be sent
	MaxRetries int `config:"max_retries"         validate:"min=-1,nonzero"`
	// BulkMaxSize max amount of events in a single batch
	BulkMaxSize int `config:"bulk_max_size"`
	// Backoff strategy configuration
	Backoff Backoff `config:"backoff"`
}

func defaultConfig() Config {
	return Config{
		TLS:         nil,
		Timeout:     30 * time.Second,
		MaxRetries:  3,
		BulkMaxSize: 50,
		Backoff:     FromGRPCBackOff(backoff.DefaultConfig),
	}
}
