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

package connection

import (
	"bytes"
	"testing"

	"github.com/elastic/beats/v8/libbeat/common"

	"github.com/stretchr/testify/assert"
)

var srvrTestInput = `/172.17.0.1:55218[0](queued=0,recved=1,sent=0)
/172.17.0.2:55218[55](queued=11,recved=22,sent=333)
/2001:0db8:85a3:0000:0000:8a2e:0370:7334:55218[0](queued=11,recved=22,sent=333)
`

func TestParser(t *testing.T) {
	conns := MetricSet{}

	mapStr, err := conns.parseCons(bytes.NewReader([]byte(srvrTestInput)))
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, len(mapStr) == 3)
	firstLine := mapStr[0]
	secondLine := mapStr[1]
	thirdLine := mapStr[2]

	firstLineClient, ok := firstLine.RootFields["client"]
	assert.True(t, ok)

	firstLineClientMap, ok := firstLineClient.(common.MapStr)
	assert.True(t, ok)

	secondLineClient, ok := secondLine.RootFields["client"]
	assert.True(t, ok)

	secondLineClientMap, ok := secondLineClient.(common.MapStr)
	assert.True(t, ok)

	thirdLineClient, ok := thirdLine.RootFields["client"]
	assert.True(t, ok)

	thirdLineClientMap, ok := thirdLineClient.(common.MapStr)
	assert.True(t, ok)

	assert.Equal(t, "172.17.0.1", firstLineClientMap["ip"])
	assert.Equal(t, "172.17.0.2", secondLineClientMap["ip"])
	assert.Equal(t, "2001:0db8:85a3:0000:0000:8a2e:0370:7334", thirdLineClientMap["ip"])

	assert.Equal(t, int64(55218), firstLineClientMap["port"])
	assert.Equal(t, int64(55218), secondLineClientMap["port"])

	assert.Equal(t, int64(0), firstLine.MetricSetFields["interest_ops"])
	assert.Equal(t, int64(55), secondLine.MetricSetFields["interest_ops"])

	assert.Equal(t, int64(0), firstLine.MetricSetFields["queued"])
	assert.Equal(t, int64(11), secondLine.MetricSetFields["queued"])

	assert.Equal(t, int64(1), firstLine.MetricSetFields["received"])
	assert.Equal(t, int64(22), secondLine.MetricSetFields["received"])

	assert.Equal(t, int64(0), firstLine.MetricSetFields["sent"])
	assert.Equal(t, int64(333), secondLine.MetricSetFields["sent"])
}
