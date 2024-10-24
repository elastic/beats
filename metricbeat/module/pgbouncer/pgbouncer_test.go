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

package pgbouncer

import (
	"fmt"
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/beats/v7/metricbeat/mb"
	mbtest "github.com/elastic/beats/v7/metricbeat/mb/testing"

	"github.com/stretchr/testify/assert"
)

func TestParseUrl(t *testing.T) {
	tests := []struct {
		Name      string
		URL       string
		Username  string
		Password  string
		Timeout   time.Duration
		Expected  string
		ExpectErr bool
	}{
		{
			Name:      "simple test",
			URL:       "postgres://host1:6432/pgbouncer",
			Expected:  "dbname='pgbouncer' host='host1' port='6432'",
			ExpectErr: false,
		},
		{
			Name:      "no port",
			URL:       "postgres://host1/pgbouncer",
			Expected:  "dbname='pgbouncer' host='host1'",
			ExpectErr: false,
		},
		{
			Name:      "user/pass in URL",
			URL:       "postgres://user:pass@host1:6432/pgbouncer",
			Expected:  "dbname='pgbouncer' host='host1' password='pass' port='6432' user='user'",
			ExpectErr: false,
		},
		{
			Name:      "user/pass in params",
			URL:       "postgres://host1:6432/pgbouncer",
			Username:  "user",
			Password:  "secret",
			Expected:  "dbname='pgbouncer' host='host1' password='secret' port='6432' user='user'",
			ExpectErr: false,
		},
		{
			Name:      "user/pass in URL take precedence",
			URL:       "postgres://user1:pass@host1:6432/pgbouncer",
			Username:  "user",
			Password:  "secret",
			Expected:  "dbname='pgbouncer' host='host1' password='pass' port='6432' user='user1'",
			ExpectErr: false,
		},
		{
			Name:      "timeout no override",
			URL:       "postgres://host1:6432/pgbouncer?connect_timeout=2",
			Expected:  "connect_timeout='2' dbname='pgbouncer' host='host1' port='6432'",
			ExpectErr: false,
		},
		{
			Name:      "timeout from param",
			URL:       "postgres://host1:6432/pgbouncer",
			Timeout:   3 * time.Second,
			Expected:  "connect_timeout='3' dbname='pgbouncer' host='host1' port='6432'",
			ExpectErr: false,
		},
		{
			Name:      "user/pass in URL take precedence, and timeout override",
			URL:       "postgres://user1:pass@host1:6432/pgbouncer?connect_timeout=2",
			Username:  "user",
			Password:  "secret",
			Timeout:   3 * time.Second,
			Expected:  "connect_timeout='3' dbname='pgbouncer' host='host1' password='pass' port='6432' user='user1'",
			ExpectErr: false,
		},
		{
			Name:      "unix socket",
			URL:       "postgresql:///pgbouncer?host=/var/lib/postgresql",
			Expected:  "dbname='pgbouncer' host='/var/lib/postgresql'",
			ExpectErr: false,
		},
		{
			Name:      "no ssl",
			URL:       "postgresql://localhost:6432/pgbouncer?sslmode=disable",
			Expected:  "dbname='pgbouncer' host='localhost' port='6432' sslmode='disable'",
			ExpectErr: false,
		},
		{
			Name:      "no scheme",
			URL:       "host1:6432/pgbouncer",
			Expected:  "dbname='pgbouncer' host='host1' port='6432'",
			ExpectErr: false,
		},
		{
			Name:      "invalid url",
			URL:       "://pgbouncer:6432",
			ExpectErr: true,
		},
		{
			Name:      "invalid schema",
			URL:       "abcd://",
			ExpectErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			mod := mbtest.NewTestModule(t, map[string]interface{}{
				"username": test.Username,
				"password": test.Password,
			})
			mod.ModConfig.Timeout = test.Timeout

			hostData, err := ParseURL(mod, test.URL)

			if test.ExpectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.Expected, hostData.URI)
			}
		})
	}
}

type MockModule struct {
	Username       string
	Password       string
	Timeout        time.Duration
	statusReporter status.StatusReporter
	statusUpdates  []statusUpdate
}
type statusUpdate struct {
	status status.Status
	msg    string
}

func (m *MockModule) Name() string {
	return "mockmodule"
}
func (m *MockModule) SetStatusReporter(statusReporter status.StatusReporter) {
	m.statusReporter = statusReporter
}
func (m *MockModule) UpdateStatus(status status.Status, msg string) {
	m.statusUpdates = append(m.statusUpdates, statusUpdate{status: status, msg: msg})
}
func (m *MockModule) Config() mb.ModuleConfig {
	return mb.ModuleConfig{
		Timeout: m.Timeout,
	}
}

func (m *MockModule) UnpackConfig(to interface{}) error {
	if m.Username == "" {
		return fmt.Errorf("simulated config error")
	}
	c, ok := to.(*struct {
		Username string `config:"username"`
		Password string `config:"password"`
	})
	if !ok {
		return fmt.Errorf("type assertion failed")
	}
	c.Username = m.Username
	c.Password = m.Password
	return nil
}

func TestParseURL_UnpackConfigError(t *testing.T) {
	mod := &MockModule{
		Username: "",
		Password: "pass",
		Timeout:  5 * time.Second,
	}
	_, err := ParseURL(mod, "postgres://localhost:5432")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "simulated config error")
}
