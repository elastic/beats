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

package module

import (
	"sync"

	"github.com/elastic/beats/v8/libbeat/beat"
)

// PublishChannels publishes the events read from each channel to the given
// publisher client. If the publisher client blocks for any reason then events
// will not be read from the given channels.
//
// This method blocks until all of the channels have been closed
// and are fully read. To stop the method immediately, close the channels and
// close the publisher client to ensure that publishing does not block. This
// may result is some events being discarded.
func PublishChannels(client beat.Client, cs ...<-chan beat.Event) {
	var wg sync.WaitGroup

	// output publishes values from c until c is closed, then calls wg.Done.
	output := func(c <-chan beat.Event) {
		defer wg.Done()
		for event := range c {
			client.Publish(event)
		}
	}

	// Start an output goroutine for each input channel in cs.
	wg.Add(len(cs))
	for _, c := range cs {
		go output(c)
	}

	wg.Wait()
}
