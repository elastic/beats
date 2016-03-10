// +build !integration

package memcache

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_UdpDatagramAddOnCompleteMessage(t *testing.T) {
	msg := &udpMessage{isComplete: true}
	buf := msg.addDatagram(&mcUdpHeader{}, []byte{1, 2, 3, 4})
	assert.Nil(t, buf)
}

func Test_UdpDatagramAddSingleDatagram(t *testing.T) {
	hdr := &mcUdpHeader{requestId: 10, seqNumber: 0, numDatagrams: 1}
	msg := newUdpMessage(hdr)
	buf := msg.addDatagram(hdr, []byte{1, 2, 3, 4})
	assert.Equal(t, 4, buf.Len())
	assert.Equal(t, []byte{1, 2, 3, 4}, buf.Bytes())
}

func Test_UdpDatagramMultiple(t *testing.T) {
	hdr := &mcUdpHeader{requestId: 10, seqNumber: 0, numDatagrams: 4}
	msg := newUdpMessage(hdr)

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
	hdr := &mcUdpHeader{requestId: 10, seqNumber: 0, numDatagrams: 4}
	msg := newUdpMessage(hdr)

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
