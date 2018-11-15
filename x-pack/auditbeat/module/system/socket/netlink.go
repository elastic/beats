// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux

package socket

import (
	"fmt"
	"os"
	"syscall"

	"github.com/elastic/gosigar/sys/linux"
)

// NetlinkSocketSubscription opens a socket and subscribes to the netlink
// multicast group for socket events.
//
// TODO: Code is very similar to elastic/gosigar/sys/linux/inetdiag.go - maybe merge?
func NetlinkSocketSubscription() (<-chan *linux.InetDiagMsg, error) {
	/*
		It's surprisingly hard to figure out the correct nl_group to subscribe to
		socket events. There doesn't seem to be any documentation containing it,
		only other code that sets it to 1, and a line in the kernel source code
		with 1 as a "magic number" (function call
		`netlink_has_listeners(uevent_sock, 1)` in function `uevent_net_broadcast_untagged`
		in linux/lib/kobject_uevent.c

		See netlink(7) for documentation.
	*/
	const netlinkGroup = 0x1

	sa := syscall.SockaddrNetlink{
		Family: syscall.AF_NETLINK,
		Groups: netlinkGroup,
	}

	socket, err := syscall.Socket(syscall.AF_NETLINK, syscall.SOCK_RAW, syscall.NETLINK_INET_DIAG)
	if err != nil {
		return nil, err
	}

	err = syscall.Bind(socket, &sa)
	if err != nil {
		return nil, err
	}

	readBuf := make([]byte, os.Getpagesize())
	socketC := make(chan *linux.InetDiagMsg)

	go keepReceiving(socket, readBuf, socketC)

	return socketC, nil
}

func keepReceiving(socket int, readBuf []byte, socketC chan *linux.InetDiagMsg) error {
	for {
		buf := readBuf
		nr, _, err := syscall.Recvfrom(socket, buf, 0)
		if err != nil {
			return err
		}
		if nr < syscall.NLMSG_HDRLEN {
			return syscall.EINVAL
		}

		buf = buf[:nr]

		messages, err := syscall.ParseNetlinkMessage(buf)
		if err != nil {
			return err
		}

		for _, m := range messages {
			if m.Header.Type == syscall.NLMSG_DONE {
				break
			}
			if m.Header.Type == syscall.NLMSG_ERROR {
				return linux.ParseNetlinkError(m.Data)
			}

			inetDiagMsg, err := linux.ParseInetDiagMsg(m.Data)
			if err != nil {
				return err
			}

			// TODO: Remove - debug
			fmt.Printf("%+v\n", inetDiagMsg)
			socketC <- inetDiagMsg
		}
	}
}
