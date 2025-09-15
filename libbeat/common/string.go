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

// NetString store the byte length of the data that follows, making it easier
// to unambiguously pass text and byte data between programs that could be
// sensitive to values that could be interpreted as delimiters or terminators
// (such as a null character).
type NetString []byte

// MarshalText exists to implement encoding.TextMarshaller interface to
// treat []byte as raw string by other encoders/serializers (e.g. JSON)
func (n NetString) MarshalText() ([]byte, error) {
	return n, nil
}
