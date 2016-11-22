// +build !linux !havepfring

package sniffer

import (
	"fmt"

	"github.com/tsg/gopacket"
)

type pfringHandle struct {
}

func newPfringHandle(device string, snaplen int, promisc bool) (*pfringHandle, error) {

	return nil, fmt.Errorf("Pfring sniffing is not compiled in")
}

func (h *pfringHandle) ReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error) {
	return data, ci, fmt.Errorf("Pfring sniffing is not compiled in")
}

func (h *pfringHandle) SetBPFFilter(expr string) (_ error) {
	return fmt.Errorf("Pfring sniffing is not compiled in")
}

func (h *pfringHandle) Enable() (_ error) {
	return fmt.Errorf("Pfring sniffing is not compiled in")
}

func (h *pfringHandle) Close() {
}
