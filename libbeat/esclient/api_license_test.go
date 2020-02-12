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

package client

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/elastic/beats/libbeat/common"

	"github.com/stretchr/testify/require"
)

func TestGetLicense(t *testing.T) {
	tests := map[string]struct {
		version        *common.Version
		resp           string
		expectedType   string
		expectedStatus string
	}{
		"v6_basic_active": {
			common.MustNewVersion("6.8.4"),
			`{"license": {"type": "basic", "status": "active"}}`,
			"basic",
			"active",
		},
		"v7_trial_active": {
			common.MustNewVersion("7.6.0"),
			`{"license": {"type": "trial", "status": "active"}}`,
			"trial",
			"active",
		},
		"v8_gold_expired": {
			common.MustNewVersion("8.0.0"),
			`{"license": {"type": "gold", "status": "expired"}}`,
			"gold",
			"expired",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var callIndex int
			server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				switch callIndex {
				case 0: // initial connection from client
					io.WriteString(rw, `{"version": {"number": "`+test.version.String()+`"}}`)
				case 1: // get license
					io.WriteString(rw, test.resp)
				}
				callIndex++
			}))
			defer server.Close()

			c, err := New(WithAddresses(server.URL))
			require.NoError(t, err)

			l, err := c.GetLicense()
			require.NoError(t, err)
			require.EqualValues(t, test.expectedStatus, l.Status)
			require.EqualValues(t, test.expectedType, l.Type)
		})
	}
}
