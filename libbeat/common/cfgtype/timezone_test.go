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

package cfgtype

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTimezoneUnpack(t *testing.T) {
	testCases := []struct {
		ZoneName string
	}{
		{"America/New_York"},
		{"Local"},
		{"+0500"},
		{"-0500"},
		{"+05:00"},
		{"-05:00"},
		{"+05"},
		{"-05"},
		{"UTC"},
	}

	for _, tc := range testCases {
		t.Run(tc.ZoneName, func(t *testing.T) {
			tz := &Timezone{}
			err := tz.Unpack(tc.ZoneName)
			require.NoError(t, err)
		})
	}
}

func TestTimezoneUnpackFixedZone(t *testing.T) {
	tz := &Timezone{}
	err := tz.Unpack("+0530")
	require.NoError(t, err)

	now := time.Time{}
	loc := tz.Location()
	offset := now.In(loc)
	offsetHour := offset.Hour()
	offsetMinute := offset.Minute()
	require.Equal(t, 5, offsetHour)
	require.Equal(t, 30, offsetMinute)
}
