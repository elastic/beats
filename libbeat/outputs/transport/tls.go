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

package transport

import (
	"time"

	"github.com/elastic/beats/v7/libbeat/common/transport"

	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/v7/libbeat/testing"
)

type TLSConfig = tlscommon.TLSConfig

type TLSVersion = tlscommon.TLSVersion

func TLSDialer(forward Dialer, config *TLSConfig, timeout time.Duration) (Dialer, error) {
	return transport.TLSDialer(forward, config, timeout)
}

func TestTLSDialer(
	d testing.Driver,
	forward Dialer,
	config *TLSConfig,
	timeout time.Duration,
) (Dialer, error) {
	return transport.TestTLSDialer(d, forward, config, timeout)
}
