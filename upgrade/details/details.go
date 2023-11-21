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
	"fmt"
	"math"
	"strings"

	"github.com/docker/go-units"
)

// DownloadRate is a float64 that can be safely marshalled to JSON
// when the value is Infinity. The rate is always in bytes/second units.
type DownloadRate float64

func (dr *DownloadRate) MarshalJSON() ([]byte, error) {
	downloadRateBytesPerSecond := float64(*dr)
	if math.IsInf(downloadRateBytesPerSecond, 0) {
		return json.Marshal("+Inf bps")
	}

	return json.Marshal(
		fmt.Sprintf("%sps", units.HumanSizeWithPrecision(downloadRateBytesPerSecond, 10)),
	)
}

func (dr *DownloadRate) UnmarshalJSON(data []byte) error {
	var downloadRateStr string
	err := json.Unmarshal(data, &downloadRateStr)
	if err != nil {
		return err
	}

	if downloadRateStr == "+Inf bps" {
		*dr = DownloadRate(math.Inf(1))
		return nil
	}

	downloadRateStr = strings.TrimSuffix(downloadRateStr, "ps")
	downloadRateBytesPerSecond, err := units.FromHumanSize(downloadRateStr)
	if err != nil {
		return err
	}

	*dr = DownloadRate(downloadRateBytesPerSecond)
	return nil
}
