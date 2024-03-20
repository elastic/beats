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

package fileout

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPathFormatString(t *testing.T) {
	tests := []struct {
		title          string
		useWindowsPath bool
		format         string
		timestamp      time.Time
		expected       string
	}{
		{
			"empty string",
			false,
			"",
			time.Time{},
			"",
		},
		{
			"no fields configured",
			false,
			"format string",
			time.Time{},
			"format string",
		},
		{
			"test timestamp formatter",
			false,
			"timestamp: %{+YYYY.MM.dd}",
			time.Date(2015, 5, 1, 20, 12, 34, 0, time.UTC),
			"timestamp: 2015.05.01",
		},
		{
			"test timestamp formatter with posix path",
			false,
			"/tmp/%{+YYYY.MM.dd}",
			time.Date(2015, 5, 1, 20, 12, 34, 0, time.UTC),
			"/tmp/2015.05.01",
		},
		{
			"test timestamp formatter with windows path",
			true,
			"C:\\tmp\\%{+YYYY.MM.dd}",
			time.Date(2015, 5, 1, 20, 12, 34, 0, time.UTC),
			"C:\\tmp\\2015.05.01",
		},
	}

	for i, test := range tests {
		t.Logf("test(%v): %v", i, test.title)
		isWindowsPath = test.useWindowsPath
		pfs := &PathFormatString{}
		err := pfs.Unpack(test.format)
		if err != nil {
			t.Error(err)
			continue
		}

		actual, err := pfs.Run(test.timestamp)

		assert.NoError(t, err)
		assert.Equal(t, test.expected, actual)
	}
}
