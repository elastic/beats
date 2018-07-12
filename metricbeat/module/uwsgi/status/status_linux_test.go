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

package status

import (
	"io/ioutil"
	"net"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
)

func TestFetchDataUnixSock(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "mb_uwsgi_status")
	assert.NoError(t, err)
	fname := tmpfile.Name()
	os.Remove(fname)

	listener, err := net.Listen("unix", fname)
	assert.NoError(t, err)
	defer os.Remove(fname)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		conn, err := listener.Accept()
		assert.NoError(t, err)

		data := testData(t)
		conn.Write(data)
		conn.Close()
		wg.Done()
	}()

	config := map[string]interface{}{
		"module":     "uwsgi",
		"metricsets": []string{"status"},
		"hosts":      []string{"unix://" + listener.Addr().String()},
	}

	f := mbtest.NewEventsFetcher(t, config)
	events, err := f.Fetch()
	assert.NoError(t, err)

	assertTestData(t, events)
	wg.Wait()
}
