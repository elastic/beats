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

package input_logfile

// Cursor allows the input to check if cursor status has been stored
// in the past and unpack the status into a custom structure.
type Cursor struct {
	resource *resource
}

func makeCursor(res *resource) Cursor {
	return Cursor{resource: res}
}

// IsNew returns true if no cursor information has been stored
// for the current Source.
func (c Cursor) IsNew() bool { return c.resource.IsNew() }

// Unpack deserialized the cursor state into to. Unpack fails if no pointer is
// given, or if the structure to points to is not compatible with the document
// stored.
func (c Cursor) Unpack(to interface{}) error {
	if c.IsNew() {
		return nil
	}
	return c.resource.UnpackCursor(to)
}
