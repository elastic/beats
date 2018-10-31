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

package elb

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func TestWatchTicks(t *testing.T) {
	lbls := []*lbListener{fakeLbl()}

	lock := sync.Mutex{}
	var startUUIDs []string
	var startLbls []*lbListener
	var stopUUIDs []string

	fetcher := newMockFetcher(lbls, nil)
	watcher := newWatcher(
		fetcher,
		time.Millisecond,
		func(uuid string, lbListener *lbListener) {
			lock.Lock()
			defer lock.Unlock()

			startUUIDs = append(startUUIDs, uuid)
			startLbls = append(startLbls, lbListener)
		},
		func(uuid string) {
			lock.Lock()
			defer lock.Unlock()

			stopUUIDs = append(stopUUIDs, uuid)
		})
	defer watcher.stop() // unnecessary, but good hygiene

	// Run through 10 ticks
	for i := 0; i < 10; i++ {
		err := watcher.once()
		require.NoError(t, err)
	}

	uuids := []string{fmt.Sprintf("%s|%s", *lbls[0].lb.LoadBalancerArn, *lbls[0].listener.ListenerArn)}

	// Test that we've seen one lbl start, but none stop
	assert.Equal(t, uuids, startUUIDs)
	assert.Len(t, stopUUIDs, 0)
	assert.Equal(t, lbls, startLbls)

	// Stop the lbl and test that we see a single stop
	// and no change to starts
	fetcher.setLbls(nil)
	watcher.once()

	assert.Equal(t, uuids, startUUIDs)
	assert.Equal(t, uuids, stopUUIDs)
}
