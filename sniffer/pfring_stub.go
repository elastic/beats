// +build !linux !havepfring

package sniffer

import (
	"fmt"

	"github.com/tsg/gopacket"
)

type PfringHandle struct {
}

func NewPfringHandle(device string, snaplen int, promisc bool) (*PfringHandle, error) {

	return nil, fmt.Errorf("Pfring sniffing is not compiled in")
}

func (h *PfringHandle) ReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error) {
	return data, ci, fmt.Errorf("Pfring sniffing is not compiled in")
}

func (h *PfringHandle) SetBPFFilter(expr string) (_ error) {
	return fmt.Errorf("Pfring sniffing is not compiled in")
}

func (h *PfringHandle) Enable() (_ error) {
	return fmt.Errorf("Pfring sniffing is not compiled in")
}

func (h *PfringHandle) Close() {
}
