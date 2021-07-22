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

package tlscommon

import (
	"github.com/elastic/beats/v7/libbeat/common"
)

// Unpack unpacks the TLS Server configuration.
func (c *ServerConfig) Unpack(cfg common.Config) error {
	const clientAuthKey = "client_authentication"
	const ca = "certificate_authorities"

	// When we have explicitely defined the `certificate_authorities` in the configuration we default
	// to `required` for the `client_authentication`, when CA is not defined we should set to `none`.
	if cfg.HasField(ca) && !cfg.HasField(clientAuthKey) {
		cfg.SetString(clientAuthKey, -1, "required")
	}
	type serverCfg ServerConfig
	var sCfg serverCfg
	if err := cfg.Unpack(&sCfg); err != nil {
		return err
	}
	*c = ServerConfig(sCfg)
	return nil
}

// Validate values the TLSConfig struct making sure certificate sure we have both a certificate and
// a key.
func (c *ServerConfig) Validate() error {
	if c.IsEnabled() {
		// c.Certificate.Validate() ensures that both a certificate and key
		// are specified, or neither are specified. For server-side TLS we
		// require both to be specified.
		if c.Certificate.Certificate == "" {
			return ErrCertificateUnspecified
		}
	}
	return c.Certificate.Validate()
}

// IsEnabled returns true if the `enable` field is set to true in the yaml.
func (c *ServerConfig) IsEnabled() bool {
	return c != nil && (c.Enabled == nil || *c.Enabled)
}
