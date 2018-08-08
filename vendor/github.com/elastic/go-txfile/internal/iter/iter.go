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

// Package iter provides functions for common array iteration strategies.
package iter

// Fn type for range based iterators.
type Fn func(len int) (begin, end int, next func(int) int)

// Forward returns limits and next function for forward iteration.
func Forward(l int) (begin, end int, next func(int) int) {
	return 0, l, func(i int) int { return i + 1 }
}

// Reversed returns limits and next function for reverse iteration.
func Reversed(l int) (begin, end int, next func(int) int) {
	return l - 1, -1, func(i int) int { return i - 1 }
}
