// Copyright 2013 Matthew Baird
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//     http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package core_test

import (
	"bytes"
	"fmt"
	"github.com/packetbeat/elastigo/api"
	"github.com/packetbeat/elastigo/core"
	"strconv"
	"time"
)

// The simplest usage of background bulk indexing
func ExampleBulkIndexer_simple() {
	indexer := core.NewBulkIndexerErrors(10, 60)
	done := make(chan bool)
	indexer.Run(done)

	indexer.Index("twitter", "user", "1", "", nil, `{"name":"bob"}`, true)

	<-done // wait forever
}

// The simplest usage of background bulk indexing with error channel
func ExampleBulkIndexer_errorchannel() {
	indexer := core.NewBulkIndexerErrors(10, 60)
	done := make(chan bool)
	indexer.Run(done)

	go func() {
		for errBuf := range indexer.ErrorChannel {
			// just blissfully print errors forever
			fmt.Println(errBuf.Err)
		}
	}()
	for i := 0; i < 20; i++ {
		indexer.Index("twitter", "user", strconv.Itoa(i), "", nil, `{"name":"bob"}`, true)
	}
	done <- true
}

// The simplest usage of background bulk indexing with error channel
func ExampleBulkIndexer_errorsmarter() {
	indexer := core.NewBulkIndexerErrors(10, 60)
	done := make(chan bool)
	indexer.Run(done)

	errorCt := 0 // use sync.atomic or something if you need
	timer := time.NewTicker(time.Minute * 3)
	go func() {
		for {
			select {
			case _ = <-timer.C:
				if errorCt < 2 {
					errorCt = 0
				}
			case _ = <-done:
				return
			}
		}
	}()

	go func() {
		for errBuf := range indexer.ErrorChannel {
			errorCt++
			fmt.Println(errBuf.Err)
			// log to disk?  db?   ????  Panic
		}
	}()
	for i := 0; i < 20; i++ {
		indexer.Index("twitter", "user", strconv.Itoa(i), "", nil, `{"name":"bob"}`, true)
	}
	done <- true // send shutdown signal
}

// The inspecting the response
func ExampleBulkIndexer_responses() {
	indexer := core.NewBulkIndexer(10)
	// Create a custom Sender Func, to allow inspection of response/error
	indexer.BulkSender = func(buf *bytes.Buffer) error {
		// @buf is the buffer of docs about to be written
		respJson, err := api.DoCommand("POST", "/_bulk", nil, buf)
		if err != nil {
			// handle it better than this
			fmt.Println(string(respJson))
		}
		return err
	}
	done := make(chan bool)
	indexer.Run(done)

	for i := 0; i < 20; i++ {
		indexer.Index("twitter", "user", strconv.Itoa(i), "", nil, `{"name":"bob"}`, true)
	}
	done <- true // send shutdown signal
}
