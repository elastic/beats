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

package http

import (
	"github.com/elastic/beats/v7/packetbeat/config"
	"github.com/elastic/beats/v7/packetbeat/protos"
	"github.com/elastic/beats/v7/packetbeat/protos/tcp"
)

type httpConfig struct {
	config.ProtocolCommon  `config:",inline"`
	SendAllHeaders         bool     `config:"send_all_headers"`
	SendHeaders            []string `config:"send_headers"`
	SplitCookie            bool     `config:"split_cookie"`
	RealIPHeader           string   `config:"real_ip_header"`
	IncludeBodyFor         []string `config:"include_body_for"`
	IncludeRequestBodyFor  []string `config:"include_request_body_for"`
	IncludeResponseBodyFor []string `config:"include_response_body_for"`
	HideKeywords           []string `config:"hide_keywords"`
	RedactAuthorization    bool     `config:"redact_authorization"`
	MaxMessageSize         int      `config:"max_message_size"`
	DecodeBody             bool     `config:"decode_body"`
	RedactHeaders          []string `config:"redact_headers"`
}

var defaultConfig = httpConfig{
	ProtocolCommon: config.ProtocolCommon{
		TransactionTimeout: protos.DefaultTransactionExpiration,
	},
	MaxMessageSize: tcp.TCPMaxDataInStream,
	DecodeBody:     true,
}
