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

package transptest

import (
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/common/transport/transptest"

	"github.com/elastic/beats/v7/libbeat/common/transport"
)

type MockServer = transptest.MockServer

type MockServerFactory = transptest.MockServerFactory

type TransportFactory = transptest.TransportFactory

func NewMockServerTCP(t *testing.T, to time.Duration, cert string, proxy *transport.ProxyConfig) *MockServer {
	return transptest.NewMockServerTCP(t, to, cert, proxy)
}

func NewMockServerTLS(t *testing.T, to time.Duration, cert string, proxy *transport.ProxyConfig) *MockServer {
	return transptest.NewMockServerTLS(t, to, cert, proxy)
}

func GenCertForTestingPurpose(t *testing.T, fileName, keyPassword string, hosts ...string) error {
	return transptest.GenCertForTestingPurpose(t, fileName, keyPassword, hosts...)
}
