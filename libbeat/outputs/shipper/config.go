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
)

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
}

func defaultConfig() Config {
	return Config{
		TLS:         nil,
		Timeout:     30 * time.Second,
		MaxRetries:  3,
		BulkMaxSize: 50,
	}
}
