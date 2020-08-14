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

package tcp

import "fmt"

type TcpConfig struct {
	Host              string `config:"host"`
	Port              int    `config:"port"`
	ReceiveBufferSize int    `config:"receive_buffer_size"`
	Delimiter         string `config:"delimiter"`
}

func defaultTcpConfig() TcpConfig {
	return TcpConfig{
		Host:              "localhost",
		Port:              2003,
		ReceiveBufferSize: 4096,
		Delimiter:         "\n",
	}
}

// Validate ensures that the configured delimiter has only one character
func (t *TcpConfig) Validate() error {
	if len(t.Delimiter) != 1 {
		return fmt.Errorf("length of delimiter is expected to be 1 but is %v", len(t.Delimiter))
	}

	return nil
}
