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

package cassandra

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/common/streambuf"
)

func TestReadInt(t *testing.T) {
	tests := []struct {
		wire uint32
		want int
	}{
		{wire: 0x00000001, want: 1},
		{wire: 0x7fffffff, want: 2147483647},
		{wire: 0x80000000, want: -2147483648},
		{wire: 0xffffffff, want: -1},
	}
	for _, tc := range tests {
		data := binary.BigEndian.AppendUint32(nil, tc.wire)
		stream := StreamDecoder{r: streambuf.New(data)}
		array := ByteArrayDecoder{Data: &data}

		assert.Equal(t, tc.want, stream.ReadInt(), "StreamDecoder.ReadInt(%#x)", tc.wire)
		assert.Equal(t, tc.want, array.ReadInt(), "ByteArrayDecoder.ReadInt(%#x)", tc.wire)
	}
}
