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

package details

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDownloadRateJSON(t *testing.T) {
	// Normal (non-infinity) download rate
	t.Run("non_infinity", func(t *testing.T) {
		downloadRate := DownloadRate(1794.7)

		data, err := json.Marshal(&downloadRate)
		require.NoError(t, err)

		var unmarshalledDownloadRate DownloadRate
		err = json.Unmarshal(data, &unmarshalledDownloadRate)
		require.NoError(t, err)
		require.Equal(t, float64(1794.7), float64(unmarshalledDownloadRate))
	})

	// Infinity download rate
	t.Run("infinity", func(t *testing.T) {
		downloadRate := DownloadRate(math.Inf(1))

		data, err := json.Marshal(&downloadRate)
		require.NoError(t, err)

		var unmarshalledDownloadRate DownloadRate
		err = json.Unmarshal(data, &unmarshalledDownloadRate)
		require.NoError(t, err)
		require.Equal(t, math.MaxFloat64, float64(unmarshalledDownloadRate))
	})
}
