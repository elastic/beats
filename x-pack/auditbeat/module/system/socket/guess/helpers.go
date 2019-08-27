// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,386 linux,amd64

package guess

import (
	"bytes"
	"fmt"
	"math/rand"
	"net"
	"time"
	"unsafe"

	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"
	"golang.org/x/sys/unix"

	"github.com/elastic/beats/libbeat/common"
)

func createSocket(bindAddr unix.SockaddrInet4) (fd int, addr unix.SockaddrInet4, err error) {
	return createSocketWithProto(unix.SOCK_STREAM, bindAddr)
}

func createSocketWithProto(proto int, bindAddr unix.SockaddrInet4) (fd int, addr unix.SockaddrInet4, err error) {
	fd, err = unix.Socket(unix.AF_INET, proto, 0)
	if err != nil {
		return -1, addr, err
	}
	if err = unix.Bind(fd, &bindAddr); err != nil {
		unix.Close(fd)
		return -1, addr, errors.Wrap(err, "bind failed")
	}
	sa, err := unix.Getsockname(fd)
	if err != nil {
		unix.Close(fd)
		return -1, addr, errors.Wrap(err, "getsockname failed")
	}
	addrptr, ok := sa.(*unix.SockaddrInet4)
	if !ok {
		unix.Close(fd)
		return -1, addr, errors.Wrap(err, "getsockname didn't return a struct sockaddr_in")
	}
	return fd, *addrptr, nil
}

func createSocket6WithProto(proto int, bindAddr unix.SockaddrInet6) (fd int, addr unix.SockaddrInet6, err error) {
	fd = -1
	fd, err = unix.Socket(unix.AF_INET6, proto, 0)
	if err != nil {
		return -1, addr, err
	}
	defer func() {
		if err != nil {
			unix.Close(fd)
		}
	}()
	if err = unix.Bind(fd, &bindAddr); err != nil {
		return -1, addr, errors.Wrap(err, "bind failed")
	}
	sa, err := unix.Getsockname(fd)
	if err != nil {
		return -1, addr, errors.Wrap(err, "getsockname failed")
	}
	addrptr, ok := sa.(*unix.SockaddrInet6)
	if !ok {
		return -1, addr, errors.Wrap(err, "getsockname didn't return a struct sockaddr_in")
	}
	return fd, *addrptr, nil
}

func alignTo(offset, align int) int {
	if offset&(align-1) != 0 {
		offset = (offset + align) & ^(align - 1)
	}
	return offset
}

func indexAligned(buf []byte, needle []byte, start, align int) int {
	n := len(needle)
	start = alignTo(start, align)
	var off, limit int
	for off, limit = start, len(buf)-n; off <= limit; off += align {
		if bytes.Equal(buf[off:off+n], needle) {
			return off
		}
	}
	return -1
}

func randomLocalIP() [4]byte {
	return [4]byte{127, uint8(rand.Intn(256)), uint8(rand.Intn(256)), uint8(1 + rand.Intn(255))}
}

func getListField(m common.MapStr, key string) ([]int, error) {
	iface, ok := m[key]
	if !ok {
		return nil, fmt.Errorf("field %s not found", key)
	}
	list, ok := iface.([]int)
	if !ok {
		return nil, fmt.Errorf("field %s is not a list", key)
	}
	if len(list) == 0 {
		return nil, fmt.Errorf("field %s not detected", key)
	}
	return list, nil
}

type ipv6loopback struct {
	fd         int
	deviceName string
	addresses  []net.IP
	ifreq      ifReq
}

type in6Ifreq struct {
	addr    [16]byte
	prefix  uint32
	ifindex int32
}

const ifnamsiz = 16

type ifReq struct {
	name    [ifnamsiz]byte
	index   int32
	padding [128]byte
}

func newIPv6Loopback() (lo ipv6loopback, err error) {
	lo.fd = -1
	devs, err := net.Interfaces()
	if err != nil {
		return lo, errors.Wrap(err, "cannot list interfaces")
	}
	for _, dev := range devs {
		addrs, err := dev.Addrs()
		if err != nil || len(dev.Name) >= ifnamsiz {
			continue
		}
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() == nil && ipnet.IP.IsLoopback() {
				lo.deviceName = dev.Name
				lo.fd, err = unix.Socket(unix.AF_INET6, unix.SOCK_DGRAM, unix.IPPROTO_IP)
				if err != nil {
					lo.fd = -1
					return lo, err
				}
				copy(lo.ifreq.name[:], dev.Name)
				lo.ifreq.name[len(dev.Name)] = 0
				_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(lo.fd), unix.SIOCGIFINDEX, uintptr(unsafe.Pointer(&lo.ifreq)))
				if errno != 0 {
					unix.Close(lo.fd)
					return lo, errors.Wrap(errno, "ioctl(SIOCGIFINDEX) failed")
				}
				return lo, nil
			}
		}
	}
	return lo, errors.New("no loopback interface detected")
}

// adds a randomly-generated address from the fd00::/8 prefix (Unique Local Address)
func (lo *ipv6loopback) addRandomAddress() (addr net.IP, err error) {
	addr = make(net.IP, 16)
	addr[0] = 0xFD
	rand.Read(addr[1:])
	var req in6Ifreq
	copy(req.addr[:], addr)
	req.ifindex = lo.ifreq.index
	req.prefix = 128
	_, _, e := unix.Syscall(unix.SYS_IOCTL, uintptr(lo.fd), unix.SIOCSIFADDR, uintptr(unsafe.Pointer(&req)))
	if e != 0 {
		return nil, e
	}
	lo.addresses = append(lo.addresses, addr)

	// wait for the added address to be available. There seems to be a small
	// delay in some systems between the time an address is added and it is
	// available to bind.
	fd, err := unix.Socket(unix.AF_INET6, unix.SOCK_DGRAM, 0)
	if err != nil {
		return addr, err
	}
	defer unix.Close(fd)
	var bindAddr unix.SockaddrInet6
	copy(bindAddr.Addr[:], addr)
	for i := 1; i < 50; i++ {
		if err = unix.Bind(fd, &bindAddr); err == nil {
			break
		}
		if errno, ok := err.(unix.Errno); !ok || errno != unix.EADDRNOTAVAIL {
			break
		}
		time.Sleep(time.Millisecond * time.Duration(i))
	}
	return addr, err
}

func (lo *ipv6loopback) Cleanup() error {
	var errs multierror.Errors
	var req in6Ifreq
	req.ifindex = lo.ifreq.index
	req.prefix = 128
	for _, addr := range lo.addresses {
		copy(req.addr[:], addr)
		_, _, e := unix.Syscall(unix.SYS_IOCTL, uintptr(lo.fd), unix.SIOCDIFADDR, uintptr(unsafe.Pointer(&req)))
		if e != 0 {
			errs = append(errs, e)
		}
	}
	if lo.fd != -1 {
		unix.Close(lo.fd)
	}
	return errs.Err()
}

// consolidate takes a list of guess results in the form of maps with []int
// values, and returns a map where for each key the value is an []int with
// the values that appeared in all the guesses.
//
// Example
// Input: [ {"A": [1,2,3,4], "B": [4, 5]}, {"A": [2,3,8], "B": [6]} ]
// Output: { "A": [2,3], "B": [] }
func consolidate(partials []common.MapStr) (result common.MapStr, err error) {
	if len(partials) == 0 {
		return nil, errors.New("empty resultset to consolidate")
	}
	result = make(common.MapStr)

	for k, v := range partials[0] {
		baseList, ok := v.([]int)
		if !ok {
			return nil, fmt.Errorf("consolidating key '%s' is not a list", k)
		}
		for idx := 1; idx < len(partials); idx++ {
			v, found := partials[idx][k]
			if !found {
				return nil, fmt.Errorf("consolidating key '%s' missing in some results", k)
			}
			list, ok := v.([]int)
			if !ok {
				return nil, fmt.Errorf("consolidating key '%s' is not always a list", k)
			}
			var newList []int
			for _, num := range baseList {
				for _, nn := range list {
					if num == nn {
						newList = append(newList, num)
						break
					}
				}
			}
			baseList = newList
			if len(baseList) == 0 {
				break
			}
		}
		result[k] = baseList
	}
	return result, nil
}

type inetClientServer struct {
	client, server, accepted int
	cliAddr                  unix.SockaddrInet4
	srvAddr                  unix.SockaddrInet4
}

// SetupTCP sets up a TCP client-server connection.
func (cs *inetClientServer) SetupTCP() (err error) {
	if cs.server, cs.srvAddr, err = createSocket(unix.SockaddrInet4{Addr: randomLocalIP()}); err != nil {
		return err
	}
	if err = unix.Listen(cs.server, 1); err != nil {
		return err
	}
	if cs.client, cs.cliAddr, err = createSocket(unix.SockaddrInet4{Addr: randomLocalIP()}); err != nil {
		return err
	}
	if err = unix.Connect(cs.client, &cs.srvAddr); err != nil {
		return err
	}
	if cs.accepted, _, err = unix.Accept(cs.server); err != nil {
		return err
	}
	return nil
}

// SetupUDP sets up a UDP client-server connection.
func (cs *inetClientServer) SetupUDP() (err error) {
	cs.accepted = -1
	cs.server, cs.srvAddr, err = createSocketWithProto(unix.SOCK_DGRAM, unix.SockaddrInet4{Addr: randomLocalIP()})
	if err != nil {
		return err
	}
	if cs.client, cs.cliAddr, err = createSocketWithProto(unix.SOCK_DGRAM, unix.SockaddrInet4{Addr: randomLocalIP()}); err != nil {
		return err
	}
	return nil
}

// Cleanup closes the sockets.
func (cs *inetClientServer) Cleanup() error {
	if cs.accepted != -1 {
		unix.Close(cs.accepted)
	}
	unix.Close(cs.server)
	unix.Close(cs.client)
	return nil
}
