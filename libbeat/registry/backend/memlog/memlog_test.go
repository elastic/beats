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

package memlog

import (
	"testing"

	"github.com/elastic/beats/v7/libbeat/registry/backend"
	"github.com/elastic/beats/v7/libbeat/registry/backend/cptest"
)

func TestComplianceDefaults(t *testing.T) {
	cptest.TestBackendCompliance(t, func(path string) (backend.Registry, error) {
		return New(Settings{
			Root: path,
		})
	})
}

func TestComplianceCheckpointAlways(t *testing.T) {
	cptest.TestBackendCompliance(t, func(path string) (backend.Registry, error) {
		return New(Settings{
			Root: path,
			Checkpoint: func(pairs, logs uint) bool {
				return true
			},
		})
	})
}

func TestComplianceLogsOnlyMode(t *testing.T) {
	cptest.TestBackendCompliance(t, func(path string) (backend.Registry, error) {
		return New(Settings{
			Root: path,
			Checkpoint: func(pairs, logs uint) bool {
				return false
			},
		})
	})
}
