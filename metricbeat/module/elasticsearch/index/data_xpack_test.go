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

// +build !integration

package index

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
)

func BenchmarkParseAPIResponse(b *testing.B) {
	// Read in large stats API response fixture
	content, err := ioutil.ReadFile("_meta/test/stats.800.bench.json")
	require.NoError(b, err)

	var indicesStats stats

	for i := 0; i < b.N; i++ {
		err = parseAPIResponse(content, &indicesStats)
		require.NoError(b, err)
	}

}
