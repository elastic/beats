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

package status

import (
	"testing"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/metricbeat/mb"

	"github.com/stretchr/testify/assert"
)

// TestConfigValidation validates that the configuration and the DSN are
// validated when the MetricSet is created.
func TestConfigValidation(t *testing.T) {
	tests := []struct {
		in  interface{}
		err string
	}{
		{
			// Missing 'hosts'
			in: map[string]interface{}{
				"module":     "mysql",
				"metricsets": []string{"status"},
			},
			err: "missing required field accessing 'hosts'",
		},
		{
			// Invalid DSN
			in: map[string]interface{}{
				"module":     "mysql",
				"metricsets": []string{"status"},
				"hosts":      []string{"127.0.0.1"},
			},
			err: "error parsing mysql host: invalid DSN: missing the slash separating the database name",
		},
		{
			// Local unix socket
			in: map[string]interface{}{
				"module":     "mysql",
				"metricsets": []string{"status"},
				"hosts":      []string{"user@unix(/path/to/socket)/"},
			},
		},
		{
			// TCP on a remote host, e.g. Amazon RDS:
			in: map[string]interface{}{
				"module":     "mysql",
				"metricsets": []string{"status"},
				"hosts":      []string{"id:password@tcp(your-amazonaws-uri.com:3306)/}"},
			},
		},
		{
			// TCP on a remote host with user/pass specified separately
			in: map[string]interface{}{
				"module":     "mysql",
				"metricsets": []string{"status"},
				"hosts":      []string{"tcp(your-amazonaws-uri.com:3306)/}"},
				"username":   "id",
				"password":   "mypass",
			},
		},
	}

	for i, test := range tests {
		c, err := common.NewConfigFrom(test.in)
		if err != nil {
			t.Fatal(err)
		}

		_, _, err = mb.NewModule(c, mb.Registry)
		if err != nil && test.err == "" {
			t.Errorf("unexpected error in testcase %d: %v", i, err)
			continue
		}
		if test.err != "" {
			if err != nil {
				assert.Contains(t, err.Error(), test.err, "testcase %d", i)
			} else {
				t.Errorf("expected error '%v' in testcase %d", test.err, i)
			}
			continue
		}
	}
}
