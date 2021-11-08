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

//go:build !integration
// +build !integration

package common

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	Timeout    time.Duration = 1 * time.Minute
	InitalSize int           = 10
)

const (
	alphaKey   = "alphaKey"
	alphaValue = "a"
	bravoKey   = "bravoKey"
	bravoValue = "b"
)

// Current time as simulated by the fakeClock function.
var (
	currentTime time.Time
	fakeClock   clock = func() time.Time {
		return currentTime
	}
)

// RemovalListener callback.
var (
	callbackKey     Key
	callbackValue   Value
	removalListener RemovalListener = func(k Key, v Value) {
		callbackKey = k
		callbackValue = v
	}
)

// Test that the removal listener is invoked with the expired key/value.
func TestExpireWithRemovalListener(t *testing.T) {
	callbackKey = nil
	callbackValue = nil
	c := newCache(Timeout, true, InitalSize, removalListener, fakeClock)
	c.Put(alphaKey, alphaValue)
	currentTime = currentTime.Add(Timeout).Add(time.Nanosecond)
	assert.Equal(t, 1, c.CleanUp())
	assert.Equal(t, alphaKey, callbackKey)
	assert.Equal(t, alphaValue, callbackValue)
}

// Test that the number of removed elements is returned by Expire.
func TestExpireWithoutRemovalListener(t *testing.T) {
	c := newCache(Timeout, true, InitalSize, nil, fakeClock)
	c.Put(alphaKey, alphaValue)
	c.Put(bravoKey, bravoValue)
	currentTime = currentTime.Add(Timeout).Add(time.Nanosecond)
	assert.Equal(t, 2, c.CleanUp())
}

func TestPutIfAbsent(t *testing.T) {
	c := newCache(Timeout, true, InitalSize, nil, fakeClock)
	oldValue := c.PutIfAbsent(alphaKey, alphaValue)
	assert.Nil(t, oldValue)
	oldValue = c.PutIfAbsent(alphaKey, bravoValue)
	assert.Equal(t, alphaValue, oldValue)
}

func TestPut(t *testing.T) {
	c := newCache(Timeout, true, InitalSize, nil, fakeClock)
	oldValue := c.Put(alphaKey, alphaValue)
	assert.Nil(t, oldValue)
	oldValue = c.Put(bravoKey, bravoValue)
	assert.Nil(t, oldValue)

	oldValue = c.Put(alphaKey, bravoValue)
	assert.Equal(t, alphaValue, oldValue)
	oldValue = c.Put(bravoKey, alphaValue)
	assert.Equal(t, bravoValue, oldValue)
}

func TestReplace(t *testing.T) {
	c := newCache(Timeout, true, InitalSize, nil, fakeClock)

	// Nil is returned when the value does not exist and no element is added.
	assert.Nil(t, c.Replace(alphaKey, alphaValue))
	assert.Equal(t, 0, c.Size())

	// alphaKey is replaced with the new value.
	assert.Nil(t, c.Put(alphaKey, alphaValue))
	assert.Equal(t, alphaValue, c.Replace(alphaKey, bravoValue))
	assert.Equal(t, 1, c.Size())
}

func TestGetUpdatesLastAccessTime(t *testing.T) {
	c := newCache(Timeout, true, InitalSize, nil, fakeClock)
	c.Put(alphaKey, alphaValue)

	currentTime = currentTime.Add(Timeout / 2)
	assert.Equal(t, alphaValue, c.Get(alphaKey))
	currentTime = currentTime.Add(Timeout / 2)
	assert.Equal(t, alphaValue, c.Get(alphaKey))
}

func TestGetDoesntUpdateLastAccessTime(t *testing.T) {
	c := newCache(Timeout, false, InitalSize, nil, fakeClock)
	c.Put(alphaKey, alphaValue)

	currentTime = currentTime.Add(Timeout - 1)
	assert.Equal(t, alphaValue, c.Get(alphaKey))
	currentTime = currentTime.Add(Timeout - 1)
	assert.Nil(t, c.Get(alphaKey))
}

func TestDeleteNonExistentKey(t *testing.T) {
	c := newCache(Timeout, true, InitalSize, nil, fakeClock)
	assert.Nil(t, c.Delete(alphaKey))
}

func TestDeleteExistingKey(t *testing.T) {
	c := newCache(Timeout, true, InitalSize, nil, fakeClock)
	c.Put(alphaKey, alphaValue)
	assert.Equal(t, alphaValue, c.Delete(alphaKey))
}

func TestDeleteExpiredKey(t *testing.T) {
	c := newCache(Timeout, true, InitalSize, nil, fakeClock)
	c.Put(alphaKey, alphaValue)
	currentTime = currentTime.Add(Timeout).Add(time.Nanosecond)
	assert.Nil(t, c.Delete(alphaKey))
}

// Test that Entries returns the non-expired map entries.
func TestEntries(t *testing.T) {
	c := newCache(Timeout, true, InitalSize, nil, fakeClock)
	c.Put(alphaKey, alphaValue)
	currentTime = currentTime.Add(Timeout).Add(time.Nanosecond)
	c.Put(bravoKey, bravoValue)
	m := c.Entries()
	assert.Equal(t, 1, len(m))
	assert.Equal(t, bravoValue, m[bravoKey])
}

// Test that Size returns a count of both expired and non-expired elements.
func TestSize(t *testing.T) {
	c := newCache(Timeout, true, InitalSize, nil, fakeClock)
	c.Put(alphaKey, alphaValue)
	currentTime = currentTime.Add(Timeout).Add(time.Nanosecond)
	c.Put(bravoKey, bravoValue)
	assert.Equal(t, 2, c.Size())
}

func TestGetExpiredValue(t *testing.T) {
	c := newCache(Timeout, true, InitalSize, nil, fakeClock)
	c.Put(alphaKey, alphaValue)
	v := c.Get(alphaKey)
	assert.Equal(t, alphaValue, v)

	currentTime = currentTime.Add(Timeout).Add(time.Nanosecond)
	v = c.Get(alphaKey)
	assert.Nil(t, v)
}

// Test that the janitor invokes CleanUp on the cache and that the
// RemovalListener is invoked during clean up.
func TestJanitor(t *testing.T) {
	keyChan := make(chan Key)
	c := newCache(Timeout, true, InitalSize, func(k Key, v Value) {
		keyChan <- k
	}, fakeClock)
	c.Put(alphaKey, alphaValue)
	currentTime = currentTime.Add(Timeout).Add(time.Nanosecond)
	c.StartJanitor(time.Millisecond)
	key := <-keyChan
	c.StopJanitor()
	assert.Equal(t, alphaKey, key)
}
