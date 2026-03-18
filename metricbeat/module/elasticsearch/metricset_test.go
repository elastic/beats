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

package elasticsearch

import (
	"testing"
)

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		backupKey string
		expected  string
		useBackup bool
	}{
		{"both set", "TEST_KEY", "BACKUP_KEY", "test_value", false},
		{"only key set", "TEST_KEY", "BACKUP_KEY", "test_value", false},
		{"only backup key set", "NOT_SET", "BACKUP_KEY", "backup_value", true},
		{"neither set", "NOT_SET", "BACKUP_KEY", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.useBackup {
				t.Setenv(tt.backupKey, tt.expected)
			} else if tt.expected != "" {
				t.Setenv(tt.key, tt.expected)
			}

			result := getEnv(tt.key, tt.backupKey)

			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}
