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

package add_process_metadata

import (
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var cacheEvictionTests = []struct {
	name string

	expire      time.Duration
	cap, effort int

	iters  int
	maxPID int
	delay  time.Duration
}{
	{
		name:   "small sparse",
		expire: time.Millisecond,
		cap:    100,
		effort: 5,
		iters:  1000,
		maxPID: 100000,
		delay:  2 * time.Millisecond,
	},
	{
		name:   "small dense",
		expire: time.Millisecond,
		cap:    100,
		effort: 5,
		iters:  1000,
		maxPID: 10,
		delay:  2 * time.Millisecond,
	},
	{
		name:   "large sparse",
		expire: time.Millisecond,
		cap:    100,
		effort: 5,
		iters:  10000,
		maxPID: 100000,
		delay:  time.Millisecond / 10,
	},
	{
		name:   "large dense",
		expire: time.Millisecond,
		cap:    100,
		effort: 5,
		iters:  10000,
		maxPID: 10,
		delay:  time.Millisecond / 10,
	},
	{
		name:   "huge sparse",
		expire: time.Millisecond,
		cap:    100,
		effort: 5,
		iters:  1000,
		maxPID: 100000,
		delay:  time.Millisecond / 100,
	},
	{
		name:   "huge dense",
		expire: time.Millisecond,
		cap:    100,
		effort: 5,
		iters:  1000,
		maxPID: 10,
		delay:  time.Millisecond / 100,
	},
}

func TestCacheEviction(t *testing.T) {
	for _, test := range cacheEvictionTests {
		rnd := rand.New(rand.NewSource(1))
		c := newProcessCache(test.expire, test.cap, test.effort, emptyProvider{})

		for i := 0; i < test.iters; i++ {
			pid := rnd.Intn(test.maxPID)
			_, err := c.GetProcessMetadata(pid)
			require.NoError(t, err)
			if len(c.cache) > test.cap {
				t.Errorf("cache overflow for %s after %d iterations", test.name, i)
				break
			}
			time.Sleep(test.delay)
		}
	}
}

type emptyProvider struct{}

func (emptyProvider) GetProcessMetadata(pid int) (*processMetadata, error) {
	return &processMetadata{pid: pid}, nil
}
