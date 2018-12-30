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

package util

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTimeout(t *testing.T) {
	// Mock monotonic time:
	fakeTimeCh := make(chan int64)
	go func() {
		fakeTime := time.Now().Unix()
		for {
			fakeTime++
			fakeTimeCh <- fakeTime
		}
	}()

	now = func() time.Time {
		return time.Unix(<-fakeTimeCh, 0)
	}

	// Blocking sleep:
	sleepCh := make(chan struct{})
	sleep = func(time.Duration) {
		<-sleepCh
	}

	test := newValueMap(1 * time.Second)

	test.Set("foo", 3.14)

	// Let cleanup do its job
	sleepCh <- struct{}{}
	sleepCh <- struct{}{}
	sleepCh <- struct{}{}

	// Check it expired
	assert.Equal(t, 0.0, test.Get("foo"))
}

func TestValueMap(t *testing.T) {
	test := newValueMap(defaultTimeout)

	// no value
	assert.Equal(t, 0.0, test.Get("foo"))

	// Set and test
	test.Set("foo", 3.14)
	assert.Equal(t, 3.14, test.Get("foo"))
}

func TestGetWithDefault(t *testing.T) {
	test := newValueMap(defaultTimeout)

	// Empty + default
	assert.Equal(t, 0.0, test.Get("foo"))
	assert.Equal(t, 3.14, test.GetWithDefault("foo", 3.14))

	// Defined value
	test.Set("foo", 38.2)
	assert.Equal(t, 38.2, test.GetWithDefault("foo", 3.14))
}

func TestContainerUID(t *testing.T) {
	assert.Equal(t, "a-b-c", ContainerUID("a", "b", "c"))
}
