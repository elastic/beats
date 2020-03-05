// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,386 linux,amd64

package state

import (
	"time"

	"golang.org/x/sys/unix"

	"github.com/elastic/go-libaudit/aucoalesce"
)

const (
	// how often to collect and report expired and terminated flows.
	reapInterval = time.Second
	// how often the state log generated (only in debug mode).
	logInterval = time.Second * 30
)

var (
	userCache  = aucoalesce.NewUserCache(5 * time.Minute)
	groupCache = aucoalesce.NewGroupCache(5 * time.Minute)
)

type FlowProto uint8

const (
	ProtoUnknown FlowProto = 0
	ProtoTCP     FlowProto = unix.IPPROTO_TCP
	ProtoUDP     FlowProto = unix.IPPROTO_UDP
)

func (p FlowProto) String() string {
	switch p {
	case ProtoTCP:
		return "tcp"
	case ProtoUDP:
		return "udp"
	}
	return "unknown"
}

type InetType uint8

const (
	InetTypeUnknown InetType = 0
	InetTypeIPv4    InetType = unix.AF_INET
	InetTypeIPv6    InetType = unix.AF_INET6
)

func (t InetType) String() string {
	switch t {
	case InetTypeIPv4:
		return "ipv4"
	case InetTypeIPv6:
		return "ipv6"
	}
	return "unknown"
}

type FlowDirection uint8

const (
	DirectionUnknown FlowDirection = iota
	DirectionInbound
	DirectionOutbound
)

// String returns the textual representation of the flowDirection.
func (d FlowDirection) String() string {
	switch d {
	case DirectionInbound:
		return "inbound"
	case DirectionOutbound:
		return "outbound"
	default:
		return "unknown"
	}
}
