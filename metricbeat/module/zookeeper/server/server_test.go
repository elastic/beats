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

package server

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

var srvrTestInput = `Zookeeper version: 3.5.5-390fe37ea45dee01bf87dc1c042b5e3dcce88653, built on 05/03/2019 12:07 GMT
Latency min/avg/max: 1/2/3
Received: 46
Sent: 45
Connections: 1
Outstanding: 0
Zxid: 0x700601132
Mode: standalone
Node count: 4
Proposal sizes last/min/max: -3/-999/-1
`

func TestParser(t *testing.T) {
	logger := logp.NewLogger("zookeeper.server")
	mapStr, versionID, err := parseSrvr(bytes.NewReader([]byte(srvrTestInput)), logger)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "2019-05-03T12:07:00Z", mapStr["version_date"])
	assert.Equal(t, "3.5.5-390fe37ea45dee01bf87dc1c042b5e3dcce88653", versionID)

	latency := mapStr["latency"].(common.MapStr)
	assert.Equal(t, int64(1), latency["min"])
	assert.Equal(t, int64(2), latency["avg"])
	assert.Equal(t, int64(3), latency["max"])

	assert.Equal(t, int64(46), mapStr["received"])
	assert.Equal(t, int64(45), mapStr["sent"])
	assert.Equal(t, int64(1), mapStr["connections"])
	assert.Equal(t, int64(0), mapStr["outstanding"])
	assert.Equal(t, "standalone", mapStr["mode"])
	assert.Equal(t, int64(4), mapStr["node_count"])

	proposalSizes := mapStr["proposal_sizes"].(common.MapStr)
	assert.Equal(t, int64(-3), proposalSizes["last"])
	assert.Equal(t, int64(-999), proposalSizes["min"])
	assert.Equal(t, int64(-1), proposalSizes["max"])

	assert.Equal(t, "0x700601132", mapStr["zxid"])
	assert.Equal(t, uint32(7), mapStr["epoch"])
	assert.Equal(t, uint32(0x601132), mapStr["count"])
}
