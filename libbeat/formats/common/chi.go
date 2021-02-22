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

package common

import (
	"math"
)

// ChiSquare calculates the chi-squared distribution of data
func ChiSquare(data []byte) float64 {
	cache := make([]float64, 256)
	for _, b := range data {
		cache[b] = cache[b] + 1
	}

	result := 0.0
	length := len(data)
	perBin := float64(length) / float64(256) // expected count per bin
	if perBin == 0 {
		return 0.0
	}
	for _, count := range cache {
		a := count - perBin
		result += (a * a) / perBin
	}
	return math.Round(result*100) / 100
}
