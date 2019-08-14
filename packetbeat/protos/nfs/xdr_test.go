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

package nfs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var testMsg = []byte{
	0x80, 0x00, 0x00, 0xe0,
	0xb5, 0x49, 0x21, 0xab,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02,
	0x00, 0x00, 0x00, 0x04,

	0x00, 0x00, 0x00, 0x0b,
	0x74, 0x65, 0x73, 0x74, 0x20, 0x73, 0x74, 0x72, 0x69, 0x6e, 0x67, 0x00,
}

func TestXdrDecoding(t *testing.T) {
	xdr := makeXDR(testMsg)

	assert.Equal(t, uint32(0x800000e0), uint32(xdr.getUInt()))
	assert.Equal(t, uint32(0xb54921ab), uint32(xdr.getUInt()))
	assert.Equal(t, uint64(2), uint64(xdr.getUHyper()))
	assert.Equal(t, uint32(4), uint32(xdr.getUInt()))
	assert.Equal(t, "test string", xdr.getString())
	assert.Equal(t, len(testMsg), xdr.size())
}
