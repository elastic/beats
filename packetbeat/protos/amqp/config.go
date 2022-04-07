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

package amqp

import (
	"github.com/elastic/beats/v8/packetbeat/config"
	"github.com/elastic/beats/v8/packetbeat/protos"
)

type amqpConfig struct {
	config.ProtocolCommon     `config:",inline"`
	ParseHeaders              bool `config:"parse_headers"`
	ParseArguments            bool `config:"parse_arguments"`
	MaxBodyLength             int  `config:"max_body_length"`
	HideConnectionInformation bool `config:"hide_connection_information"`
}

var defaultConfig = amqpConfig{
	ProtocolCommon: config.ProtocolCommon{
		TransactionTimeout: protos.DefaultTransactionExpiration,
	},
	ParseHeaders:              true,
	ParseArguments:            true,
	MaxBodyLength:             1000,
	HideConnectionInformation: true,
}
