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

package journalread

import (
	"testing"
)

func TestMode_Unpack(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		tests := map[string]SeekMode{
			"head":   SeekHead,
			"tail":   SeekTail,
			"cursor": SeekCursor,
		}

		for str, want := range tests {
			t.Run(str, func(t *testing.T) {
				var m SeekMode
				if err := m.Unpack(str); err != nil {
					t.Fatal(err)
				}

				if m != want {
					t.Errorf("wrong mode, expected %v, got %v", want, m)
				}
			})
		}
	})

	t.Run("failing", func(t *testing.T) {
		cases := []string{"invalid", "", "unknown"}

		for _, str := range cases {
			t.Run(str, func(t *testing.T) {
				var m SeekMode
				if err := m.Unpack(str); err == nil {
					t.Errorf("an error was expected, got %v", m)
				}
			})
		}
	})
}
