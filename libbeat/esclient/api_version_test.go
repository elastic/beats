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

func TestGetVersion(t *testing.T) {
	v760 := common.MustNewVersion("7.6.0")

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		io.WriteString(rw, `{"version": {"number": "`+v760.String()+`"}}`)
	}))
	defer server.Close()

	c, err := New(WithAddresses(server.URL))
	require.NoError(t, err)
	require.EqualValues(t, v760, c.GetVersion())
}
