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

package channel

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/menderesk/beats/v7/libbeat/beat"
	"github.com/menderesk/beats/v7/libbeat/tests/resources"
)

type dummyOutletter struct {
	closed bool
	c      chan struct{}
}

func (o *dummyOutletter) OnEvent(event beat.Event) bool {
	return true
}

func (o *dummyOutletter) Close() error {
	o.closed = true
	close(o.c)
	return nil
}

func (o *dummyOutletter) Done() <-chan struct{} {
	return o.c
}

func TestCloseOnSignal(t *testing.T) {
	goroutines := resources.NewGoroutinesChecker()
	defer goroutines.Check(t)

	o := &dummyOutletter{c: make(chan struct{})}
	sig := make(chan struct{})
	CloseOnSignal(o, sig)
	close(sig)
}

func TestCloseOnSignalClosed(t *testing.T) {
	goroutines := resources.NewGoroutinesChecker()
	defer goroutines.Check(t)

	o := &dummyOutletter{c: make(chan struct{})}
	sig := make(chan struct{})
	c := CloseOnSignal(o, sig)
	c.Close()
}

func TestSubOutlet(t *testing.T) {
	goroutines := resources.NewGoroutinesChecker()
	defer goroutines.Check(t)

	o := &dummyOutletter{c: make(chan struct{})}
	so := SubOutlet(o)
	so.Close()
	assert.False(t, o.closed)
}

func TestCloseOnSignalSubOutlet(t *testing.T) {
	goroutines := resources.NewGoroutinesChecker()
	defer goroutines.Check(t)

	o := &dummyOutletter{c: make(chan struct{})}
	c := CloseOnSignal(SubOutlet(o), make(chan struct{}))
	o.Close()
	c.Close()
}
