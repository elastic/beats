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

package tcp

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestValidate(t *testing.T) {
	type testCfg struct {
		name    string
		cfg     Config
		wantErr error
	}

	tests := []testCfg{
		{
			name: "ok",
			cfg: Config{
				Host: "localhost:9000",
			},
		},
		{
			name:    "nohost",
			wantErr: ErrMissingHostPort,
		},
		{
			name: "invalidnetwork",
			cfg: Config{
				Host:    "localhost:9000",
				Network: "foo",
			},
			wantErr: ErrInvalidNetwork,
		},
	}

	for _, network := range []string{networkTCP, networkTCP4, networkTCP6} {
		tests = append(tests, testCfg{
			name: "network_" + network,
			cfg: Config{
				Host:    "localhost:9000",
				Network: network,
			},
		})
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
			diff := cmp.Diff(tc.wantErr, err, cmpopts.EquateErrors())
			if diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
