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

package postgresql

import (
	"testing"
	"time"

	mbtest "github.com/elastic/beats/v8/metricbeat/mb/testing"

	"github.com/stretchr/testify/assert"
)

func TestParseUrl(t *testing.T) {
	tests := []struct {
		Name     string
		URL      string
		Username string
		Password string
		Timeout  time.Duration
		Expected string
	}{
		{
			Name:     "simple test",
			URL:      "postgres://host1:5432",
			Expected: "host='host1' port='5432'",
		},
		{
			Name:     "no port",
			URL:      "postgres://host1",
			Expected: "host='host1'",
		},
		{
			Name:     "user/pass in URL",
			URL:      "postgres://user:pass@host1:5432",
			Expected: "host='host1' password='pass' port='5432' user='user'",
		},
		{
			Name:     "user/pass in params",
			URL:      "postgres://host1:5432",
			Username: "user",
			Password: "secret",
			Expected: "host='host1' password='secret' port='5432' user='user'",
		},
		{
			Name:     "user/pass in URL take precedence",
			URL:      "postgres://user1:pass@host1:5432",
			Username: "user",
			Password: "secret",
			Expected: "host='host1' password='pass' port='5432' user='user1'",
		},
		{
			Name:     "timeout no override",
			URL:      "postgres://host1:5432?connect_timeout=2",
			Expected: "connect_timeout='2' host='host1' port='5432'",
		},
		{
			Name:     "timeout from param",
			URL:      "postgres://host1:5432",
			Timeout:  3 * time.Second,
			Expected: "connect_timeout='3' host='host1' port='5432'",
		},
		{
			Name:     "user/pass in URL take precedence, and timeout override",
			URL:      "postgres://user1:pass@host1:5432?connect_timeout=2",
			Username: "user",
			Password: "secret",
			Timeout:  3 * time.Second,
			Expected: "connect_timeout='3' host='host1' password='pass' port='5432' user='user1'",
		},
		{
			Name:     "unix socket",
			URL:      "postgresql:///dbname?host=/var/lib/postgresql",
			Expected: "dbname='dbname' host='/var/lib/postgresql'",
		},
	}

	for _, test := range tests {
		mod := mbtest.NewTestModule(t, map[string]interface{}{
			"username": test.Username,
			"password": test.Password,
		})
		mod.ModConfig.Timeout = test.Timeout
		hostData, err := ParseURL(mod, test.URL)
		if err != nil {
			t.Error(err)
			continue
		}

		assert.Equal(t, test.Expected, hostData.URI, test.Name)
	}
}
