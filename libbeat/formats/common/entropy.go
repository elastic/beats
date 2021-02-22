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

import "math"

// Entropy calculates the entropy of data
func Entropy(data []byte) float64 {
	cache := make(map[byte]int)
	for _, b := range data {
		if found, ok := cache[b]; ok {
			cache[b] = found + 1
		} else {
			cache[b] = 1
		}
	}

	result := 0.0
	length := len(data)
	for _, count := range cache {
		frequency := float64(count) / float64(length)
		result -= frequency * math.Log2(frequency)
	}
	return math.Round(result*100) / 100
}
