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

package monitorstate

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func requireMSStatusCount(t *testing.T, ms *State, status StateStatus, count int) {
	if status == StatusUp {
		requireMSCounts(t, ms, count, 0)
	} else if status == StatusDown {
		requireMSCounts(t, ms, 0, count)
	} else {
		panic("can only check up or down statuses")
	}
}

func requireMSCounts(t *testing.T, ms *State, up int, down int) {
	require.Equal(t, up+down, ms.Checks, "expected %d total checks, got %d (%d up / %d down)", up+down, ms.Checks, ms.Up, ms.Down)
	require.Equal(t, up, ms.Up, "expected %d up checks, got %d", up, ms.Up)
	require.Equal(t, down, ms.Down, "expected %d down checks, got %d", down, ms.Down)
}
