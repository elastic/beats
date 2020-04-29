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

package generator

import "strings"

var generators = map[string]IDGenerator{
	"elasticsearch": ESTimeBasedUUIDGenerator(),
}

// IDGenerator implementors know how to generate and return a new ID
type IDGenerator interface {
	NextID() string
}

// Factory takes as input the type of ID to generate and returns the
// generator of that ID type.
func Factory(val string) (IDGenerator, error) {
	typ := strings.ToLower(val)
	g, found := generators[typ]
	if !found {
		return nil, makeErrUnknownType(val)
	}

	return g, nil
}

// Exists returns whether the given type of ID generator exists.
func Exists(val string) bool {
	typ := strings.ToLower(val)
	_, found := generators[typ]
	return found
}
