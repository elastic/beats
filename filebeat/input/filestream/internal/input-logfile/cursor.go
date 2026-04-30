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

import (
	"github.com/elastic/go-concert/unison"
)

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

// AllEventsPublished returns true if there are no pending operations
// on this cursor, which means all events have been published.
//
// Owners of an resource can be active inputs or pending update operations
// (not yet written to disk).
//
// The harvester locks this resource (see `startHarvester`) and only releases
// when it is shutdown. So if there is only one 'owner' of this resource, it
// is safe to assume it is the harvester, therefore all events have been
// published.
//
// There is a test ensuring this behaviour, see TestCursorAllEventsPublished
func (c Cursor) AllEventsPublished() bool {
	pending := c.resource.pending.Load()
	if pending == 1 || pending == 0 {
		return true
	}

	return false
}

// NewCursorForTest returns a Cursor for testing. It MUST NOT be used
// on production code. `key` and `pending` will be directly set into the
// underlying resource
//
// If `pending` is -1, then the default is applied, which results in the
// same state that `filestream.Run` expects.
//
// The resource associated with this cursor is created with the same
// logic `states.unsafeFind` uses.
func NewCursorForTest(key string, offset int64, pending int) Cursor {
	res := resource{
		stored: false,
		key:    key,
		lock:   unison.MakeMutex(),
		cursor: struct {
			Offset int64 `json:"offset" struct:"offset"`
		}{
			Offset: offset,
		},
	}

	if pending == -1 {
		res.Retain()
	} else {
		//nolint:gosec // 'pending' is always positive
		res.pending.Store(uint64(pending))
	}

	res.internalState.TTL = -1

	return makeCursor(&res)
}
