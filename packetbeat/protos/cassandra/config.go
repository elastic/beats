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

package cassandra

import (
	"fmt"

	"github.com/elastic/beats/packetbeat/config"
	"github.com/elastic/beats/packetbeat/protos"

	gocql "github.com/elastic/beats/packetbeat/protos/cassandra/internal/gocql"
)

type cassandraConfig struct {
	config.ProtocolCommon `config:",inline"`
	SendRequestHeader     bool            `config:"send_request_header"`
	SendResponseHeader    bool            `config:"send_response_header"`
	Compressor            string          `config:"compressor"`
	OPsIgnored            []gocql.FrameOp `config:"ignored_ops"`
}

var (
	defaultConfig = cassandraConfig{
		ProtocolCommon: config.ProtocolCommon{
			TransactionTimeout: protos.DefaultTransactionExpiration,
			SendRequest:        true,
			SendResponse:       true,
		},
		SendRequestHeader:  true,
		SendResponseHeader: true,
	}
)

func (c *cassandraConfig) Validate() error {
	if !(c.Compressor == "" || c.Compressor == "snappy") {
		return fmt.Errorf("invalid compressor config: %s, only snappy supported", c.Compressor)
	}
	return nil
}
