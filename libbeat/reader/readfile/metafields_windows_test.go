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

package readfile

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/common/file"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

type winTestInfo struct {
	testFileInfo
	idxhi uint32
	idxlo uint32
	vol   uint32
}

func createTestFileInfo() file.ExtendedFileInfo {
	return file.ExtendFileInfo(&winTestInfo{
		testFileInfo: testFileInfo{
			name: "filename",
			size: 42,
			time: time.Now(),
		},
		idxhi: 100,
		idxlo: 200,
		vol:   300,
	})
}

func checkFields(t *testing.T, expected, actual mapstr.M) {
	t.Helper()

	idxhi, err := actual.GetValue(idxhiKey)
	require.NoError(t, err)
	require.Equal(t, "100", idxhi)
	err = actual.Delete(idxhiKey)
	require.NoError(t, err)

	idxlo, err := actual.GetValue(idxloKey)
	require.NoError(t, err)
	require.Equal(t, "200", idxlo)
	err = actual.Delete(idxloKey)
	require.NoError(t, err)

	vol, err := actual.GetValue(volKey)
	require.NoError(t, err)
	require.Equal(t, "300", vol)
	err = actual.Delete(volKey)
	require.NoError(t, err)

	require.Equal(t, expected, actual)
}
