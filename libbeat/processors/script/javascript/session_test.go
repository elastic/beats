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

package javascript

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/menderesk/beats/v7/libbeat/beat"
	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/logp"

	"github.com/stretchr/testify/assert"
)

func TestSessionTagOnException(t *testing.T) {
	const script = `throw "this tags the event";`

	p, err := NewFromConfig(Config{
		Source:         header + script + footer,
		TagOnException: defaultConfig().TagOnException,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	evt, err := p.Run(testEvent())
	assert.Error(t, err)

	tags, _ := evt.GetValue("tags")
	assert.Equal(t, []string{"_js_exception"}, tags)
}

func TestSessionScriptParams(t *testing.T) {
	t.Run("register method is optional", func(t *testing.T) {
		_, err := NewFromConfig(Config{
			Source: header + footer,
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("register required for params", func(t *testing.T) {
		_, err := NewFromConfig(Config{
			Source: header + footer,
			Params: map[string]interface{}{
				"threshold": 42,
			},
		}, nil)
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "params were provided")
		}
	})

	t.Run("register params", func(t *testing.T) {
		const script = `
			function register(params) {
				if (params["threshold"] !== 42) {
					throw "invalid threshold";
				}
			}

			function process(event) {}
		`
		_, err := NewFromConfig(Config{
			Source: script,
			Params: map[string]interface{}{
				"threshold": 42,
			},
		}, nil)
		assert.NoError(t, err)
	})
}

func TestSessionTestFunction(t *testing.T) {
	const script = `
		var fail = false;

		function register(params) {
			fail = params["fail"];
		}

		function process(event) {
			if (fail) {
				throw "intentional failure";
			}
			event.Put("hello", "world");
 			return event;
		}

		function test() {
			var event = process(new Event({"hello": "earth"}));

			if (event.fields.hello !== "world") {
				throw "invalid hello world";
 			}
		}
	`

	t.Run("test method is optional", func(t *testing.T) {
		_, err := NewFromConfig(Config{
			Source: header + footer,
		}, nil)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("test success", func(t *testing.T) {
		_, err := NewFromConfig(Config{
			Source: script,
			Params: map[string]interface{}{
				"fail": false,
			},
		}, nil)
		assert.NoError(t, err)
	})

	t.Run("test failure", func(t *testing.T) {
		_, err := NewFromConfig(Config{
			Source: script,
			Params: map[string]interface{}{
				"fail": true,
			},
		}, nil)
		assert.Error(t, err)
	})
}

func TestSessionTimeout(t *testing.T) {
	logp.TestingSetup()

	const runawayLoop = `
		while (!evt.fields.stop) {
			evt.Put("hello", "world");			
		}
    `

	p, err := NewFromConfig(Config{
		Source:         header + runawayLoop + footer,
		Timeout:        500 * time.Millisecond,
		TagOnException: "_js_exception",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	evt := &beat.Event{
		Fields: common.MapStr{
			"stop": false,
		},
	}

	// Execute and expect a timeout.
	evt, err = p.Run(evt)
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), timeoutError)

		tags, _ := evt.GetValue("tags")
		assert.Equal(t, []string{"_js_exception"}, tags)

		errorMessage, _ := evt.GetValue("error.message")
		assert.Contains(t, errorMessage, timeoutError)
	}

	// Verify that any internal runtime interrupt state has been cleared.
	evt.PutValue("stop", true)
	_, err = p.Run(evt)
	assert.NoError(t, err)
}

func TestSessionParallel(t *testing.T) {
	const script = `
		evt.Put("host.name", "workstation");			
    `

	p, err := NewFromConfig(Config{
		Source:         header + script + footer,
		TagOnException: "_js_exception",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	const goroutines = 10
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for ctx.Err() == nil {
				evt := &beat.Event{
					Fields: common.MapStr{
						"host": common.MapStr{"name": "computer"},
					},
				}
				_, err := p.Run(evt)
				assert.NoError(t, err)
			}
		}()
	}

	time.AfterFunc(time.Second, cancel)
	wg.Wait()
}
