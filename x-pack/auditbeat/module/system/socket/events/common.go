// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,386 linux,amd64

package events

import (
	"fmt"
	"syscall"

	"github.com/elastic/beats/v7/x-pack/auditbeat/tracing"
)

const (
	// This compensates the size argument of udp_sendmsg which is only
	// UDP payload. 28 is the size of an IPv4 header (no options) + UDP header.
	minIPv4UdpPacketSize = 28

	// Same for udpv6_sendmsg.
	// 40 is the size of an IPv6 header (no extensions) + UDP header.
	minIPv6UdpPacketSize = 48
)

func header(meta tracing.Metadata) string {
	return fmt.Sprintf("%d probe=%d pid=%d tid=%d",
		meta.Timestamp,
		meta.EventID,
		meta.PID,
		meta.TID)
}

func kernErrorDesc(retval int32) string {
	switch {
	case retval < 0:
		errno := syscall.Errno(uintptr(0 - retval))
		return fmt.Sprintf("failed errno=%d (%s)", errno, errno.Error())
	case retval == 0:
		return "ok"
	default:
		return fmt.Sprintf("ok (value=%d)", retval)
	}
}
