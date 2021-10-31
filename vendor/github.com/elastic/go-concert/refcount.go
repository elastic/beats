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

package concert

import (
	"sync"

	"github.com/elastic/go-concert/atomic"
)

// RefCount is an atomic reference counter. It can be used to track a shared
// resource it's lifetime and execute an action once it is clear the resource is
// not needed anymore.
//
// The zero value of RefCount is already in a valid state, which can be
// Released already.
type RefCount struct {
	count atomic.Uint32

	errMux sync.Mutex
	err    error

	Action  func(err error)
	OnError func(old, new error) error
}

// refCountFree indicates when a RefCount.Release shall return true.  It's
// chosen such that the zero value of RefCount is a valid value which will
// return true if Release is called without calling Retain before.
const refCountFree uint32 = ^uint32(0)
const refCountOops uint32 = refCountFree - 1

// Retain increases the ref count.
func (c *RefCount) Retain() {
	if c.count.Inc() == 0 {
		panic("retaining released ref count")
	}
}

// Release decreases the reference count. It returns true, if the reference count
// has reached a 'free' state.
// Releasing a reference count in a free state will trigger a panic.
// If an Action is configured, then this action will be run once the
// refcount becomes free.
func (c *RefCount) Release() bool {
	switch c.count.Dec() {
	case refCountFree:
		if c.Action != nil {
			c.Action(c.err)
		}
		return true
	case refCountOops:
		panic("ref count released too often")
	default:
		return false
	}
}

// Err returns the current error stored by the reference counter.
func (c *RefCount) Err() error {
	c.errMux.Lock()
	defer c.errMux.Unlock()
	return c.err
}

// Fail adds an error to the reference counter.
// OnError will be called if configured, so to compute the actual error.
// If OnError is not configured, the first error reported will be stored by
// the reference counter only.
//
// Fail releases the reference counter.
func (c *RefCount) Fail(err error) bool {
	// use dummy function to handle the error, ensuring errMux.Unlock will be
	// executed before the call to Release or in case OnError panics.
	func() {
		c.errMux.Lock()
		defer c.errMux.Unlock()
		if c.OnError != nil {
			c.err = c.OnError(c.err, err)
		} else if c.err == nil {
			c.err = err
		}
	}()

	return c.Release()
}
