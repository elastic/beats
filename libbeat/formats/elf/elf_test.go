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

package elf

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBinaries(t *testing.T) {
	generate := os.Getenv("GENERATE") == "1"
	binaries := []string{
		"hello-linux",
	}
	for _, binary := range binaries {
		t.Run(binary, func(t *testing.T) {
			f, err := os.Open("../fixtures/elf/" + binary)
			require.NoError(t, err)
			defer f.Close()

			info, err := Parse(f)
			require.NoError(t, err)

			expectedFile := "../fixtures/elf/" + binary + ".fingerprint"
			if generate {
				data, err := json.MarshalIndent(info, "", "  ")
				require.NoError(t, err)
				require.NoError(t, ioutil.WriteFile(expectedFile, data, 0644))
			} else {
				fixture, err := os.Open(expectedFile)
				require.NoError(t, err)
				defer fixture.Close()
				expected, err := ioutil.ReadAll(fixture)
				require.NoError(t, err)

				data, err := json.Marshal(info)
				require.NoError(t, err)
				require.JSONEq(t, string(expected), string(data))
			}
		})
	}
}
