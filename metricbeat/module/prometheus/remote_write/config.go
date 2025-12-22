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

package remote_write

import "github.com/elastic/elastic-agent-libs/transport/tlscommon"

const (
	// DefaultMaxCompressedBodyBytes is the default maximum size of compressed request body (50MB)
	DefaultMaxCompressedBodyBytes int64 = 50 * 1024 * 1024
	// DefaultMaxDecodedBodyBytes is the default maximum size of decoded request body (250MB)
	DefaultMaxDecodedBodyBytes int64 = 250 * 1024 * 1024
)

type Config struct {
	MetricsCount           bool                    `config:"metrics_count"`
	Host                   string                  `config:"host"`
	Port                   int                     `config:"port"`
	TLS                    *tlscommon.ServerConfig `config:"ssl"`
	MaxCompressedBodyBytes int64                   `config:"max_compressed_body_bytes"`
	MaxDecodedBodyBytes    int64                   `config:"max_decoded_body_bytes"`
}

func defaultConfig() Config {
	return Config{
		Host:                   "localhost",
		Port:                   9201,
		MaxCompressedBodyBytes: DefaultMaxCompressedBodyBytes,
		MaxDecodedBodyBytes:    DefaultMaxDecodedBodyBytes,
	}
}
