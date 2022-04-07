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

//go:build !integration
// +build !integration

package mysql

import (
	"testing"
	"time"

	mbtest "github.com/elastic/beats/v8/metricbeat/mb/testing"

	"github.com/stretchr/testify/assert"
)

func TestParseDSN(t *testing.T) {
	const query = "?readTimeout=10s&timeout=10s&writeTimeout=10s"

	var tests = []struct {
		host     string
		username string
		password string
		uri      string
	}{
		{"", "", "", "tcp(127.0.0.1:3306)/" + query},
		{"", "root", "secret", "root:secret@tcp(127.0.0.1:3306)/" + query},
		{"unix(/tmp/mysql.sock)/", "root", "", "root@unix(/tmp/mysql.sock)/" + query},
		{"tcp(127.0.0.1:3306)/", "", "", "tcp(127.0.0.1:3306)/" + query},
		{"tcp(127.0.0.1:3306)/", "root", "", "root@tcp(127.0.0.1:3306)/" + query},
		{"tcp(127.0.0.1:3306)/", "root", "secret", "root:secret@tcp(127.0.0.1:3306)/" + query},
	}

	for _, test := range tests {
		c := map[string]interface{}{
			"username": test.username,
			"password": test.password,
		}
		mod := mbtest.NewTestModule(t, c)
		mod.ModConfig.Timeout = 10 * time.Second

		hostData, err := ParseDSN(mod, test.host)
		if err != nil {
			t.Error(err)
			continue
		}

		assert.Equal(t, test.uri, hostData.URI)
		if test.username != "" {
			assert.NotContains(t, hostData.SanitizedURI, test.username)
		}
		if test.password != "" {
			assert.NotContains(t, hostData.SanitizedURI, test.password)
		}
	}
}
