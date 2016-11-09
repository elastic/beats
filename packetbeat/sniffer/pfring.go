// +build linux,havepfring

package sniffer

import (
	"fmt"

	"github.com/tsg/gopacket"
	"github.com/tsg/gopacket/pfring"
)

type pfringHandle struct {
	Ring *pfring.Ring
}

func newPfringHandle(device string, snaplen int, promisc bool) (*pfringHandle, error) {

	var h pfringHandle
	var err error

	if device == "any" {
		return nil, fmt.Errorf("Pfring sniffing doesn't support 'any' as interface")
	}

	var flags pfring.Flag

	if promisc {
		flags = pfring.FlagPromisc
	}

	h.Ring, err = pfring.NewRing(device, uint32(snaplen), flags)

	return &h, err
}

func (h *pfringHandle) ReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error) {
	return h.Ring.ReadPacketData()
}

func (h *pfringHandle) SetBPFFilter(expr string) (_ error) {
	return h.Ring.SetBPFFilter(expr)
}

func (h *pfringHandle) Enable() (_ error) {
	return h.Ring.Enable()
}

func (h *pfringHandle) Close() {
	h.Ring.Close()
}
