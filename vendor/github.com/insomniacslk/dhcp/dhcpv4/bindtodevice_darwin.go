// +build darwin

package dhcpv4

import (
	"net"
	"syscall"
)

func BindToInterface(fd int, ifname string) error {
	iface, err := net.InterfaceByName(ifname)
	if err != nil {
		return err
	}
	return syscall.SetsockoptInt(fd, syscall.IPPROTO_IP, syscall.IP_BOUND_IF, iface.Index)
}
