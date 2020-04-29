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

package resources

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGoroutinesChecker(t *testing.T) {
	block := make(chan struct{})
	defer close(block)

	cases := []struct {
		title   string
		test    func()
		timeout time.Duration
		fail    bool
	}{
		{
			title: "no goroutines",
			test:  func() {},
		},
		{
			title: "fast goroutine",
			test: func() {
				started := make(chan struct{})
				go func() {
					started <- struct{}{}
				}()
				<-started
			},
		},
		/* Skipped due to flakyness: https://github.com/elastic/beats/issues/12692
		{
			title: "blocked goroutine",
			test: func() {
				started := make(chan struct{})
				go func() {
					started <- struct{}{}
					<-block
				}()
				<-started
			},
			timeout: 500 * time.Millisecond,
			fail:    true,
		},
		*/
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			goroutines := NewGoroutinesChecker()
			if c.timeout > 0 {
				goroutines.FinalizationTimeout = c.timeout
			}
			c.test()
			err := goroutines.check(t)
			if c.fail {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
