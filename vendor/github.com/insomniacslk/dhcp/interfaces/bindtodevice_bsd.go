// +build aix freebsd openbsd netbsd darwin

package interfaces

import (
	"net"
	"syscall"
)

// BindToInterface emulates linux's SO_BINDTODEVICE option for a socket by using
// IP_RECVIF.
func BindToInterface(fd int, ifname string) error {
	iface, err := net.InterfaceByName(ifname)
	if err != nil {
		return err
	}
	return syscall.SetsockoptInt(fd, syscall.IPPROTO_IP, syscall.IP_RECVIF, iface.Index)
}
