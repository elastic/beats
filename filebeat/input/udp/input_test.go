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

package udp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProcNetUDP(t *testing.T) {
	t.Run("with_match", func(t *testing.T) {
		rx, drops, err := procNetUDP("testdata/proc_net_udp.txt", []string{"2508640A:1BBE"})
		if err != nil {
			t.Fatal(err)
		}
		assert.EqualValues(t, 1, rx)
		assert.EqualValues(t, 2, drops)
	})

	t.Run("without_match", func(t *testing.T) {
		_, _, err := procNetUDP("testdata/proc_net_udp.txt", []string{"FOO:BAR", "BAR:BAZ"})
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "entry not found")
		}
	})
}
