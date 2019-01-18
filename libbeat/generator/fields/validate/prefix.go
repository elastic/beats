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

package validate

// Prefix helps construct a field name in dotted format.
type Prefix string

// Append adds a new component to the prefix.
func (p Prefix) Append(key string) Prefix {
	if len(p) > 0 {
		return Prefix(p.String() + "." + key)
	}
	return Prefix(key)
}

// Key adds a new component to the prefix and returns its string representation.
func (p Prefix) Key(key string) string {
	return p.Append(key).String()
}

// String returns the string representation of a prefix.
func (p Prefix) String() string {
	return string(p)
}
