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

package connections

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

var srvrTestInput = `/172.17.0.1:55218[0](queued=0,recved=1,sent=0)
/172.17.0.2:55218[0](queued=11,recved=22,sent=333)

`

func TestParser(t *testing.T) {
	mapStr, err := parseCons(bytes.NewReader([]byte(srvrTestInput)))
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, int64(22), mapStr["received"])
}
