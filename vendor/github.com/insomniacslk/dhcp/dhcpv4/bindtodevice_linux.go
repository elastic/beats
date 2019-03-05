// +build linux

package dhcpv4

import (
	"syscall"
)

func BindToInterface(fd int, ifname string) error {
	return syscall.BindToDevice(fd, ifname)
}
