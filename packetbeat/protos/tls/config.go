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

package tls

import (
	"github.com/elastic/beats/packetbeat/config"
	"github.com/elastic/beats/packetbeat/protos"
)

type tlsConfig struct {
	config.ProtocolCommon  `config:",inline"`
	SendCertificates       bool     `config:"send_certificates"`
	IncludeRawCertificates bool     `config:"include_raw_certificates"`
	Fingerprints           []string `config:"fingerprints"`
}

var (
	defaultConfig = tlsConfig{
		ProtocolCommon: config.ProtocolCommon{
			TransactionTimeout: protos.DefaultTransactionExpiration,
		},
		SendCertificates:       true,
		IncludeRawCertificates: false,
		Fingerprints:           []string{"sha1"},
	}
)
