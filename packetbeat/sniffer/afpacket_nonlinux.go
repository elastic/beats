// +build !linux

package sniffer

import (
	"fmt"
	"time"

	"github.com/tsg/gopacket"
)

type afpacketHandle struct {
}

func newAfpacketHandle(device string, snaplen int, blockSize int, numBlocks int,
	timeout time.Duration) (*afpacketHandle, error) {

	return nil, fmt.Errorf("Afpacket MMAP sniffing is only available on Linux")
}

func (h *afpacketHandle) ReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error) {
	return data, ci, fmt.Errorf("Afpacket MMAP sniffing is only available on Linux")
}

func (h *afpacketHandle) SetBPFFilter(expr string) (_ error) {
	return fmt.Errorf("Afpacket MMAP sniffing is only available on Linux")
}

func (h *afpacketHandle) Close() {
}
