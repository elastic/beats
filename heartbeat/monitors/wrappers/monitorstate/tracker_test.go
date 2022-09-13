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

func TestTrackerRecord(t *testing.T) {
	mst := NewTracker(NilStateLoader, true)
	ms := mst.RecordStatus(TestSf, StatusUp)
	require.Equal(t, StatusUp, ms.Status)
	requireMSStatusCount(t, ms, StatusUp, 1)

	for i := 0; i < FlappingThreshold; i++ {
		_ = mst.RecordStatus(TestSf, StatusDown)
		ms = mst.RecordStatus(TestSf, StatusUp)
	}
	require.Equal(t, StatusFlapping, ms.Status)
	requireMSCounts(t, ms, FlappingThreshold+1, FlappingThreshold)

	// Restore stable state
	for i := 0; i < FlappingThreshold; i++ {
		_ = mst.RecordStatus(TestSf, StatusDown)
	}

	ms = mst.RecordStatus(TestSf, StatusDown)
	require.Equal(t, StatusDown, ms.Status)
	requireMSStatusCount(t, ms, StatusDown, FlappingThreshold-1)
}

func TestTrackerRecordFlappingDisabled(t *testing.T) {
	mst := NewTracker(NilStateLoader, false)
	ms := mst.RecordStatus(TestSf, StatusUp)
	require.Equal(t, StatusUp, ms.Status)
	requireMSStatusCount(t, ms, StatusUp, 1)

	for i := 0; i < FlappingThreshold; i++ {
		_ = mst.RecordStatus(TestSf, StatusDown)
		ms = mst.RecordStatus(TestSf, StatusUp)
	}

	// with flapping disabled it only shows as up
	require.Equal(t, StatusUp, ms.Status)
	requireMSCounts(t, ms, 1, 0)

	ms = mst.RecordStatus(TestSf, StatusDown)
	require.Equal(t, StatusDown, ms.Status)
	requireMSStatusCount(t, ms, StatusDown, 1)
}
