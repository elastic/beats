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

// +build !integration

package memcache

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_UdpDatagramAddOnCompleteMessage(t *testing.T) {
	msg := &udpMessage{isComplete: true}
	buf := msg.addDatagram(&mcUDPHeader{}, []byte{1, 2, 3, 4})
	assert.Nil(t, buf)
}

func Test_UdpDatagramAddSingleDatagram(t *testing.T) {
	hdr := &mcUDPHeader{requestID: 10, seqNumber: 0, numDatagrams: 1}
	msg := newUDPMessage(hdr)
	buf := msg.addDatagram(hdr, []byte{1, 2, 3, 4})
	assert.Equal(t, 4, buf.Len())
	assert.Equal(t, []byte{1, 2, 3, 4}, buf.Bytes())
}

func Test_UdpDatagramMultiple(t *testing.T) {
	hdr := &mcUDPHeader{requestID: 10, seqNumber: 0, numDatagrams: 4}
	msg := newUDPMessage(hdr)

	buf := msg.addDatagram(hdr, []byte{1, 2})
	assert.Nil(t, buf)

	hdr.seqNumber = 2
	buf = msg.addDatagram(hdr, []byte{5, 6})
	assert.Nil(t, buf)

	hdr.seqNumber = 1
	buf = msg.addDatagram(hdr, []byte{3, 4})
	assert.Nil(t, buf)

	hdr.seqNumber = 3
	buf = msg.addDatagram(hdr, []byte{7, 8})
	assert.NotNil(t, buf)

	assert.Equal(t, 8, buf.Len())
	assert.Equal(t, []byte{1, 2, 3, 4, 5, 6, 7, 8}, buf.Bytes())
}

func Test_UdpDatagramMultipleDups(t *testing.T) {
	hdr := &mcUDPHeader{requestID: 10, seqNumber: 0, numDatagrams: 4}
	msg := newUDPMessage(hdr)

	buf := msg.addDatagram(hdr, []byte{1, 2})
	assert.Nil(t, buf)

	hdr.seqNumber = 2
	buf = msg.addDatagram(hdr, []byte{5, 6})
	assert.Nil(t, buf)

	hdr.seqNumber = 0
	buf = msg.addDatagram(hdr, []byte{1, 2})
	assert.Nil(t, buf)

	hdr.seqNumber = 1
	buf = msg.addDatagram(hdr, []byte{3, 4})
	assert.Nil(t, buf)

	hdr.seqNumber = 2
	buf = msg.addDatagram(hdr, []byte{5, 6})
	assert.Nil(t, buf)

	hdr.seqNumber = 3
	buf = msg.addDatagram(hdr, []byte{7, 8})
	assert.NotNil(t, buf)

	hdr.seqNumber = 3
	tmp := msg.addDatagram(hdr, []byte{7, 8})
	assert.Nil(t, tmp)

	assert.Equal(t, 8, buf.Len())
	assert.Equal(t, []byte{1, 2, 3, 4, 5, 6, 7, 8}, buf.Bytes())
}
