// +build linux

package sniffer

import (
	"time"

	"github.com/tsg/gopacket"
	"github.com/tsg/gopacket/afpacket"
)

type AfpacketHandle struct {
	TPacket *afpacket.TPacket
}

func NewAfpacketHandle(device string, snaplen int, block_size int, num_blocks int,
	timeout time.Duration) (*AfpacketHandle, error) {

	var h AfpacketHandle
	var err error

	if device == "any" {
		h.TPacket, err = afpacket.NewTPacket(
			afpacket.OptFrameSize(snaplen),
			afpacket.OptBlockSize(block_size),
			afpacket.OptNumBlocks(num_blocks),
			afpacket.OptPollTimeout(timeout))
	} else {
		h.TPacket, err = afpacket.NewTPacket(
			afpacket.OptInterface(device),
			afpacket.OptFrameSize(snaplen),
			afpacket.OptBlockSize(block_size),
			afpacket.OptNumBlocks(num_blocks),
			afpacket.OptPollTimeout(timeout))
	}

	return &h, err
}

func (h *AfpacketHandle) ReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error) {
	return h.TPacket.ReadPacketData()
}

func (h *AfpacketHandle) SetBPFFilter(expr string) (_ error) {
	return h.TPacket.SetBPFFilter(expr)
}

func (h *AfpacketHandle) Close() {
	h.TPacket.Close()
}
