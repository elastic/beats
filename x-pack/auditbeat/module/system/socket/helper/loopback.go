// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build (linux && 386) || (linux && amd64)
// +build linux,386 linux,amd64

package helper

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
	"time"
	"unsafe"

	"github.com/joeshaw/multierror"
	"golang.org/x/sys/unix"
)

// IPv6Loopback is a helper to add random link-local IPv6 addresses to the
// loopback interface.
type IPv6Loopback struct {
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

// NewIPv6Loopback detects the loopback interface and creates an IPv6Loopback
// to add and remove link-local IPv6 addresses.
func NewIPv6Loopback() (lo IPv6Loopback, err error) {
	lo.fd = -1
	devs, err := net.Interfaces()
	if err != nil {
		return lo, fmt.Errorf("cannot list interfaces: %w", err)
	}
	for _, dev := range devs {
		addrs, err := dev.Addrs()
		if err != nil || len(dev.Name) >= ifnamsiz {
			continue
		}
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.IsLoopback() {
				lo.deviceName = dev.Name
				lo.fd, err = unix.Socket(unix.AF_INET6, unix.SOCK_DGRAM, unix.IPPROTO_IP)
				if err != nil {
					lo.fd = -1
					return lo, fmt.Errorf("ipv6 socket failed: %w", err)
				}
				copy(lo.ifreq.name[:], dev.Name)
				lo.ifreq.name[len(dev.Name)] = 0
				_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(lo.fd), unix.SIOCGIFINDEX, uintptr(unsafe.Pointer(&lo.ifreq)))
				if errno != 0 {
					unix.Close(lo.fd)
					return lo, fmt.Errorf("ioctl(SIOCGIFINDEX) failed: %w", errno)
				}
				return lo, nil
			}
		}
	}
	return lo, errors.New("no loopback interface detected")
}

// AddRandomAddress adds a randomly-generated address
// from the fd00::/8 prefix (Unique Local Address)
func (lo *IPv6Loopback) AddRandomAddress() (addr net.IP, err error) {
	addr = make(net.IP, 16)
	addr[0] = 0xFD
	rand.Read(addr[1:])
	var req in6Ifreq
	copy(req.addr[:], addr)
	req.ifindex = lo.ifreq.index
	req.prefix = 128
	_, _, e := unix.Syscall(unix.SYS_IOCTL, uintptr(lo.fd), unix.SIOCSIFADDR, uintptr(unsafe.Pointer(&req)))
	if e != 0 {
		return nil, fmt.Errorf("ioctl SIOCSIFADDR failed: %w", e)
	}
	lo.addresses = append(lo.addresses, addr)

	// wait for the added address to be available. There seems to be a small
	// delay in some systems between the time an address is added and it is
	// available to bind.
	fd, err := unix.Socket(unix.AF_INET6, unix.SOCK_DGRAM, 0)
	if err != nil {
		return addr, fmt.Errorf("socket ipv6 dgram failed: %w", err)
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
	if err != nil {
		err = fmt.Errorf("bind failed: %w", err)
	}
	return addr, err
}

// Cleanup removes the addresses registered to this loopback.
func (lo *IPv6Loopback) Cleanup() error {
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
