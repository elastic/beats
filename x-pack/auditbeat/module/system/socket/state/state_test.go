// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,386 linux,amd64

package state

import (
	"encoding/binary"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system/socket/common"
	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system/socket/dns"
	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system/socket/events"
	"github.com/elastic/beats/v7/x-pack/auditbeat/tracing"
	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sys/unix"
)

type logWrapper testing.T

func (l *logWrapper) Errorf(format string, args ...interface{}) {
	l.Logf("error: "+format, args)
}

func (l *logWrapper) Warnf(format string, args ...interface{}) {
	l.Logf("warning: "+format, args)
}

func (l *logWrapper) Infof(format string, args ...interface{}) {
	l.Logf("info: "+format, args)
}

func (l *logWrapper) Debugf(format string, args ...interface{}) {
	l.Logf("debug: "+format, args)
}

func TestTCPConnWithProcess(t *testing.T) {
	const (
		localIP            = "192.168.33.10"
		remoteIP           = "172.19.12.13"
		localPort          = 38842
		remotePort         = 443
		sock       uintptr = 0xff1234
	)
	st := makeState(nil, (*logWrapper)(t), time.Second, time.Second, time.Second, 0, time.Second)
	lPort, rPort := be16(localPort), be16(remotePort)
	lAddr, rAddr := ipv4(localIP), ipv4(remoteIP)
	evs := []common.Event{
		execveCall(meta(1234, 1234, 1), []string{"/usr/bin/curl", "https://example.net/", "-o", "/tmp/site.html"}),
		&events.CommitCredsCall{Meta: meta(1234, 1234, 2), UID: 501, GID: 20, EUID: 501, EGID: 20},
		&events.ExecveReturn{Meta: meta(1234, 1234, 2), Retval: 1234},
		&events.InetCreateCall{Meta: meta(1234, 1235, 5), Proto: 0},
		&events.SockInitDataCall{Meta: meta(1234, 1235, 5), Socket: sock},
		&events.TCPv4ConnectCall{Meta: meta(1234, 1235, 8), Socket: sock, RAddr: rAddr, RPort: rPort},
		&events.IPLocalOutCall{
			Meta:   meta(1234, 1235, 8),
			Socket: sock,
			Size:   20,
			LAddr:  lAddr,
			LPort:  lPort,
			RAddr:  rAddr,
			RPort:  rPort,
		},
		&events.TCPConnectReturn{Meta: meta(1234, 1235, 9), Retval: 0},
		&events.TCPv4DoRcvCall{
			Meta:   meta(0, 0, 12),
			Socket: sock,
			Size:   12,
			LAddr:  lAddr,
			LPort:  lPort,
			RAddr:  rAddr,
			RPort:  rPort,
		},
		&events.InetReleaseCall{Meta: meta(0, 0, 15), Socket: sock},
		&events.TCPv4DoRcvCall{
			Meta:   meta(0, 0, 17),
			Socket: sock,
			Size:   7,
			LAddr:  lAddr,
			LPort:  lPort,
			RAddr:  rAddr,
			RPort:  rPort,
		},
		&events.DoExitCall{Meta: meta(1234, 1234, 18)},
	}
	feedEvents(evs, st, t)
	st.CleanUp()
	flows, err := getFlows(st.PopFlows(), all)
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, flows, 1)
	flow := flows[0]
	t.Log("read flow", flow)
	for field, expected := range map[string]interface{}{
		"source.ip":           localIP,
		"source.port":         localPort,
		"source.packets":      uint64(1),
		"source.bytes":        uint64(20),
		"client.ip":           localIP,
		"client.port":         localPort,
		"destination.ip":      remoteIP,
		"destination.port":    remotePort,
		"destination.packets": uint64(2),
		"destination.bytes":   uint64(19),
		"server.ip":           remoteIP,
		"server.port":         remotePort,
		"network.direction":   "outbound",
		"network.transport":   "tcp",
		"network.type":        "ipv4",
		"process.pid":         1234,
		"process.name":        "curl",
		"user.id":             "501",
	} {
		if !assertValue(t, flow, expected, field) {
			t.Fatal("expected value not found")
		}
	}
}

func TestTCPConnWithProcessTimeouts(t *testing.T) {
	const (
		localIP            = "192.168.33.10"
		remoteIP           = "172.19.12.13"
		localPort          = 38842
		remotePort         = 443
		sock       uintptr = 0xff1234
	)
	st := makeState(nil, (*logWrapper)(t), time.Second, 0, time.Second, 0, time.Second)
	lPort, rPort := be16(localPort), be16(remotePort)
	lAddr, rAddr := ipv4(localIP), ipv4(remoteIP)
	evs := []common.Event{
		execveCall(meta(1234, 1234, 1), []string{"/usr/bin/curl", "https://example.net/", "-o", "/tmp/site.html"}),
		&events.CommitCredsCall{Meta: meta(1234, 1234, 2), UID: 501, GID: 20, EUID: 501, EGID: 20},
		&events.ExecveReturn{Meta: meta(1234, 1234, 2), Retval: 1234},
		&events.InetCreateCall{Meta: meta(1234, 1235, 5), Proto: 0},
		&events.SockInitDataCall{Meta: meta(1234, 1235, 5), Socket: sock},
		&events.TCPv4ConnectCall{Meta: meta(1234, 1235, 8), Socket: sock, RAddr: rAddr, RPort: rPort},
		&events.IPLocalOutCall{
			Meta:   meta(1234, 1235, 8),
			Socket: sock,
			Size:   20,
			LAddr:  lAddr,
			LPort:  lPort,
			RAddr:  rAddr,
			RPort:  rPort,
		},
		&events.TCPConnectReturn{Meta: meta(1234, 1235, 9), Retval: 0},
		&events.TCPv4DoRcvCall{
			Meta:   meta(0, 0, 12),
			Socket: sock,
			Size:   12,
			LAddr:  lAddr,
			LPort:  lPort,
			RAddr:  rAddr,
			RPort:  rPort,
		},
	}
	feedEvents(evs, st, t)
	st.CleanUp()
	flows, err := getFlows(st.PopFlows(), all)
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, flows, 1)
	flow := flows[0]
	for field, expected := range map[string]interface{}{
		"source.ip":           localIP,
		"source.port":         localPort,
		"source.packets":      uint64(1),
		"source.bytes":        uint64(20),
		"client.ip":           localIP,
		"client.port":         localPort,
		"destination.ip":      remoteIP,
		"destination.port":    remotePort,
		"destination.packets": uint64(1),
		"destination.bytes":   uint64(12),
		"server.ip":           remoteIP,
		"server.port":         remotePort,
		"network.direction":   "outbound",
		"network.transport":   "tcp",
		"network.type":        "ipv4",
		"process.pid":         1234,
		"process.name":        "curl",
		"user.id":             "501",
	} {
		if !assertValue(t, flow, expected, field) {
			t.Fatal("expected value not found")
		}
	}

	evs = []common.Event{
		&events.InetReleaseCall{Meta: meta(0, 0, 15), Socket: sock},
		&events.TCPv4DoRcvCall{
			Meta:   meta(0, 0, 17),
			Socket: sock,
			Size:   7,
			LAddr:  lAddr,
			LPort:  lPort,
			RAddr:  rAddr,
			RPort:  rPort,
		},
		&events.DoExitCall{Meta: meta(1234, 1234, 18)},
	}
	feedEvents(evs, st, t)
	st.CleanUp()
	flows, err = getFlows(st.PopFlows(), all)
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, flows, 1)
	// we have a truncated flow with no directionality,
	// so just report what we have
	flow = flows[0]
	for field, expected := range map[string]interface{}{
		"source.ip":         localIP,
		"source.port":       localPort,
		"client.ip":         localIP,
		"client.port":       localPort,
		"destination.ip":    remoteIP,
		"destination.port":  remotePort,
		"server.ip":         remoteIP,
		"server.port":       remotePort,
		"network.direction": "unknown",
		"network.transport": "tcp",
		"network.type":      "ipv4",
	} {
		if !assertValue(t, flow, expected, field) {
			t.Fatal("expected value not found")
		}
	}
}

func TestUDPOutgoingSinglePacketWithProcess(t *testing.T) {
	const (
		localIP            = "192.168.33.10"
		remoteIP           = "172.19.12.13"
		localPort          = 38842
		remotePort         = 53
		sock       uintptr = 0xff1234
	)
	st := makeState(nil, (*logWrapper)(t), time.Second, time.Second, time.Second, 0, time.Second)
	lPort, rPort := be16(localPort), be16(remotePort)
	lAddr, rAddr := ipv4(localIP), ipv4(remoteIP)
	evs := []common.Event{
		execveCall(meta(1234, 1234, 1), []string{"/usr/bin/exfil-udp"}),
		&events.CommitCredsCall{Meta: meta(1234, 1234, 2), UID: 501, GID: 20, EUID: 501, EGID: 20},
		&events.ExecveReturn{Meta: meta(1234, 1234, 2), Retval: 1234},
		&events.InetCreateCall{Meta: meta(1234, 1235, 5), Proto: 0},
		&events.SockInitDataCall{Meta: meta(1234, 1235, 5), Socket: sock},
		&events.UDPSendmsgCall{
			Meta:     meta(1234, 1235, 6),
			Socket:   sock,
			Size:     123,
			LAddr:    lAddr,
			AltRAddr: rAddr,
			LPort:    lPort,
			AltRPort: rPort,
		},
		&events.InetReleaseCall{Meta: meta(1234, 1235, 17), Socket: sock},
		&events.DoExitCall{Meta: meta(1234, 1234, 18)},
	}
	feedEvents(evs, st, t)
	st.CleanUp()
	flows, err := getFlows(st.PopFlows(), all)
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, flows, 1)
	flow := flows[0]
	t.Log("read flow", flow)
	for field, expected := range map[string]interface{}{
		"source.ip":           localIP,
		"source.port":         localPort,
		"source.packets":      uint64(1),
		"source.bytes":        uint64(151),
		"client.ip":           localIP,
		"client.port":         localPort,
		"destination.ip":      remoteIP,
		"destination.port":    remotePort,
		"destination.packets": uint64(0),
		"destination.bytes":   uint64(0),
		"server.ip":           remoteIP,
		"server.port":         remotePort,
		"network.direction":   "outbound",
		"network.transport":   "udp",
		"network.type":        "ipv4",
		"process.pid":         1234,
		"process.name":        "exfil-udp",
		"user.id":             "501",
	} {
		assertValue(t, flow, expected, field)
	}
}

func TestUDPIncomingSinglePacketWithProcess(t *testing.T) {
	const (
		localIP            = "192.168.33.10"
		remoteIP           = "172.19.12.13"
		localPort          = 38842
		remotePort         = 53
		sock       uintptr = 0xff1234
	)
	st := makeState(nil, (*logWrapper)(t), time.Second, time.Second, time.Second, 0, time.Second)
	lPort, rPort := be16(localPort), be16(remotePort)
	lAddr, rAddr := ipv4(localIP), ipv4(remoteIP)
	var packet [256]byte
	var ipHdr, udpHdr uint16 = 2, 64
	packet[ipHdr] = 0x45
	tracing.MachineEndian.PutUint32(packet[ipHdr+12:], rAddr)
	tracing.MachineEndian.PutUint16(packet[udpHdr:], rPort)
	evs := []common.Event{
		execveCall(meta(1234, 1234, 1), []string{"/usr/bin/exfil-udp"}),
		&events.CommitCredsCall{Meta: meta(1234, 1234, 2), UID: 501, GID: 20, EUID: 501, EGID: 20},
		&events.ExecveReturn{Meta: meta(1234, 1234, 2), Retval: 1234},
		&events.InetCreateCall{Meta: meta(1234, 1235, 5), Proto: 0},
		&events.SockInitDataCall{Meta: meta(1234, 1235, 5), Socket: sock},
		&events.UDPQueueRcvSkbCall{
			Meta:   meta(1234, 1235, 5),
			Socket: sock,
			Size:   123,
			LAddr:  lAddr,
			LPort:  lPort,
			IPHdr:  ipHdr,
			UDPHdr: udpHdr,
			Packet: packet,
		},
		&events.InetReleaseCall{Meta: meta(1234, 1235, 17), Socket: sock},
		&events.DoExitCall{Meta: meta(1234, 1234, 18)},
	}
	feedEvents(evs, st, t)
	st.CleanUp()
	flows, err := getFlows(st.PopFlows(), all)
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, flows, 1)
	flow := flows[0]
	t.Log("read flow", flow)
	for field, expected := range map[string]interface{}{
		"source.ip":           remoteIP,
		"source.port":         remotePort,
		"source.packets":      uint64(1),
		"source.bytes":        uint64(151),
		"client.ip":           remoteIP,
		"client.port":         remotePort,
		"destination.ip":      localIP,
		"destination.port":    localPort,
		"destination.packets": uint64(0),
		"destination.bytes":   uint64(0),
		"server.ip":           localIP,
		"server.port":         localPort,
		"network.direction":   "inbound",
		"network.transport":   "udp",
		"network.type":        "ipv4",
		"process.pid":         1234,
		"process.name":        "exfil-udp",
		"user.id":             "501",
	} {
		assertValue(t, flow, expected, field)
	}
}

type noDNSResolution struct{}

func (noDNSResolution) ResolveIP(pid uint32, ip net.IP) (domain string, found bool) {
	return "", false
}

type dnsTestCase struct {
	found      bool
	proc       *common.Process
	ip, domain string
}

type dnsTestCases []dnsTestCase

func (c dnsTestCases) Run(t *testing.T) {
	for idx, test := range c {
		msg := fmt.Sprintf("test entry #%d : %+v", idx, test)
		domain, found := test.proc.ResolveIP(net.ParseIP(test.ip))
		assert.Equal(t, test.found, found, msg)
		assert.Equal(t, test.domain, domain, msg)
	}
}

func TestDNSTracker(t *testing.T) {
	const infiniteExpiration = time.Hour * 3
	local1 := net.UDPAddr{IP: net.ParseIP("192.168.0.2"), Port: 55555}
	local2 := net.UDPAddr{IP: net.ParseIP("192.168.0.2"), Port: 55556}
	trV4 := dns.Transaction{
		TXID:      1234,
		Client:    local1,
		Server:    net.UDPAddr{IP: net.ParseIP("8.8.8.8"), Port: 53},
		Domain:    "example.net",
		Addresses: []net.IP{net.ParseIP("192.0.2.12"), net.ParseIP("192.0.2.13")},
	}
	trV6 := dns.Transaction{
		TXID:      1235,
		Client:    local2,
		Server:    net.UDPAddr{IP: net.ParseIP("8.8.8.8"), Port: 53},
		Domain:    "example.com",
		Addresses: []net.IP{net.ParseIP("2001:db8::1111"), net.ParseIP("2001:db8::2222")},
	}
	t.Run("transaction before register", func(t *testing.T) {
		proc1 := &common.Process{PID: 123}
		proc2 := &common.Process{PID: 124}
		tracker := newDNSTracker(infiniteExpiration)
		tracker.addTransaction(trV4)
		tracker.addTransaction(trV6)
		tracker.registerEndpoint(&local1, proc1)
		tracker.registerEndpoint(&local2, proc1)
		dnsTestCases{
			{true, proc1, "192.0.2.12", "example.net"},
			{true, proc1, "192.0.2.13", "example.net"},
			{true, proc1, "2001:db8::1111", "example.com"},
			{true, proc1, "2001:db8::2222", "example.com"},
			{false, proc2, "192.0.2.12", ""},
			{false, proc2, "2001:db8::2222", ""},
			{false, proc1, "192.168.0.2", ""},
			{false, proc1, "2001:db8::3333", ""},
		}.Run(t)
	})
	t.Run("transaction after register", func(t *testing.T) {
		proc1 := &common.Process{PID: 123}
		proc2 := &common.Process{PID: 124}
		tracker := newDNSTracker(infiniteExpiration)
		tracker.registerEndpoint(&local1, proc1)
		tracker.registerEndpoint(&local2, proc1)
		tracker.addTransaction(trV4)
		tracker.addTransaction(trV6)
		dnsTestCases{
			{true, proc1, "192.0.2.12", "example.net"},
			{true, proc1, "192.0.2.13", "example.net"},
			{true, proc1, "2001:db8::1111", "example.com"},
			{true, proc1, "2001:db8::2222", "example.com"},
			{false, proc2, "192.0.2.12", ""},
			{false, proc2, "2001:db8::2222", ""},
			{false, proc1, "192.168.0.2", ""},
			{false, proc1, "2001:db8::3333", ""},
		}.Run(t)
	})
	t.Run("unknown local endpoint", func(t *testing.T) {
		proc1 := &common.Process{PID: 123}
		proc2 := &common.Process{PID: 124}
		tracker := newDNSTracker(infiniteExpiration)
		tracker.registerEndpoint(&local1, proc1)
		tracker.addTransaction(trV4)
		tracker.addTransaction(trV6)
		dnsTestCases{
			{true, proc1, "192.0.2.12", "example.net"},
			{true, proc1, "192.0.2.13", "example.net"},
			{false, proc1, "2001:db8::1111", ""},
			{false, proc1, "2001:db8::2222", ""},
			{false, proc2, "192.0.2.12", ""},
			{false, proc2, "2001:db8::2222", ""},
			{false, proc1, "192.168.0.2", ""},
			{false, proc1, "2001:db8::3333", ""},
		}.Run(t)
	})
	t.Run("expiration", func(t *testing.T) {
		proc1 := &common.Process{PID: 123}
		tracker := newDNSTracker(10 * time.Millisecond)
		tracker.addTransaction(trV4)
		tracker.addTransaction(trV6)
		time.Sleep(time.Millisecond * 50)
		tracker.registerEndpoint(&local1, proc1)
		tracker.registerEndpoint(&local2, proc1)
		dnsTestCases{
			{false, proc1, "192.0.2.12", ""},
			{false, proc1, "192.0.2.13", ""},
			{false, proc1, "2001:db8::1111", ""},
			{false, proc1, "2001:db8::2222", ""},
		}.Run(t)
	})
	t.Run("same IP different domains", func(t *testing.T) {
		proc1 := &common.Process{PID: 123}
		proc2 := &common.Process{PID: 124}
		trV4alt := dns.Transaction{
			TXID:      1234,
			Client:    local2,
			Server:    net.UDPAddr{IP: net.ParseIP("8.8.8.8"), Port: 53},
			Domain:    "example.com",
			Addresses: []net.IP{net.ParseIP("192.0.2.12"), net.ParseIP("192.0.2.13")},
		}
		tracker := newDNSTracker(infiniteExpiration)
		tracker.addTransaction(trV4)
		tracker.addTransaction(trV4alt)
		tracker.registerEndpoint(&local1, proc1)
		tracker.registerEndpoint(&local2, proc2)
		dnsTestCases{
			{true, proc1, "192.0.2.12", "example.net"},
			{true, proc2, "192.0.2.12", "example.com"},
		}.Run(t)
	})
}

func TestUDPSendMsgAltLogic(t *testing.T) {
	const expectedIPv4 = "6 probe=0 pid=1234 tid=1235 udp_sendmsg(sock=0x0, size=0, 10.11.12.13:1010 -> 10.20.30.40:1234)"
	const expectedIPv6 = "6 probe=0 pid=1234 tid=1235 udpv6_sendmsg(sock=0x0, size=0, [fddd::bebe]:1010 -> [fddd::cafe]:1234)"
	t.Run("ipv4 non-connected", func(t *testing.T) {
		ev := &events.UDPSendmsgCall{
			Meta:     meta(1234, 1235, 6),
			LAddr:    ipv4("10.11.12.13"),
			LPort:    be16(1010),
			RAddr:    ipv4("10.20.30.40"),
			RPort:    be16(1234),
			AltRAddr: ipv4("192.168.255.255"),
			AltRPort: be16(555),
			SIPtr:    0x7fffffff,
			SIAF:     unix.AF_INET,
		}
		assert.Equal(t, expectedIPv4, ev.String())
	})
	t.Run("ipv4 connected", func(t *testing.T) {
		ev := &events.UDPSendmsgCall{
			Meta:     meta(1234, 1235, 6),
			LAddr:    ipv4("10.11.12.13"),
			LPort:    be16(1010),
			RAddr:    ipv4("192.168.255.255"),
			RPort:    be16(555),
			AltRAddr: ipv4("10.20.30.40"),
			AltRPort: be16(1234),
		}
		assert.Equal(t, expectedIPv4, ev.String())
	})
	t.Run("ipv6 non-connected", func(t *testing.T) {
		ev := &events.UDPv6SendmsgCall{
			Meta:     meta(1234, 1235, 6),
			LPort:    be16(1010),
			RPort:    be16(1234),
			AltRPort: be16(555),
			SI6Ptr:   0x7fffffff,
			SI6AF:    unix.AF_INET6,
		}
		ev.LAddrA, ev.LAddrB = ipv6("fddd::bebe")
		ev.RAddrA, ev.RAddrB = ipv6("fddd::cafe")
		ev.AltRAddrA, ev.AltRAddrB = ipv6("fddd::bad:bad")
		assert.Equal(t, expectedIPv6, ev.String())
	})

	t.Run("ipv6 connected", func(t *testing.T) {
		ev := &events.UDPv6SendmsgCall{
			Meta:     meta(1234, 1235, 6),
			LPort:    be16(1010),
			RPort:    be16(555),
			AltRPort: be16(1234),
		}
		ev.LAddrA, ev.LAddrB = ipv6("fddd::bebe")
		ev.RAddrA, ev.RAddrB = ipv6("fddd::bad:bad")
		ev.AltRAddrA, ev.AltRAddrB = ipv6("fddd::cafe")
		assert.Equal(t, expectedIPv6, ev.String())
	})
}

func execveCall(meta tracing.Metadata, args []string) *events.ExecveCall {
	event := &events.ExecveCall{
		Meta: meta,
	}
	lim := len(args)
	if lim > common.MaxProgArgs {
		lim = common.MaxProgArgs
	}
	for i := 0; i < lim; i++ {
		event.Ptrs[i] = 1
	}
	if lim < len(args) {
		event.Ptrs[lim] = 1
	}
	switch lim {
	case 5:
		copyCString(event.Param4[:], []byte(args[4]))
		fallthrough
	case 4:
		copyCString(event.Param3[:], []byte(args[3]))
		fallthrough
	case 3:
		copyCString(event.Param2[:], []byte(args[2]))
		fallthrough
	case 2:
		copyCString(event.Param1[:], []byte(args[1]))
		fallthrough
	case 1:
		copyCString(event.Param0[:], []byte(args[0]))
	case 0:
		return nil
	}
	event.Path = event.Param0
	return event
}

func meta(pid uint32, tid uint32, timestamp uint64) tracing.Metadata {
	return tracing.Metadata{
		Timestamp: timestamp,
		TID:       tid,
		PID:       pid,
	}
}

func copyCString(dst []byte, src []byte) {
	copy(dst, src)
	if len(src) < len(dst) {
		dst[len(src)] = 0
	} else {
		dst[len(dst)-1] = 0
	}
}

func ipv4(ip string) uint32 {
	netIP := net.ParseIP(ip).To4()
	if netIP == nil {
		panic("bad ip")
	}
	return tracing.MachineEndian.Uint32(netIP)
}

func ipv6(ip string) (hi uint64, lo uint64) {
	netIP := net.ParseIP(ip).To16()
	if netIP == nil {
		panic("bad ip")
	}
	return tracing.MachineEndian.Uint64(netIP[:]), tracing.MachineEndian.Uint64(netIP[8:])
}

func feedEvents(evs []common.Event, st *State, t *testing.T) {
	for idx, ev := range evs {
		t.Logf("Delivering event %d: %s", idx, ev.String())
		ev.Update(st)
	}
}

func getFlows(flows []*common.Flow, filter func(*common.Flow) bool) (evs []beat.Event, err error) {
	var errs multierror.Errors
	for _, flow := range flows {
		if !flow.IsValid() {
			errs = append(errs, errors.New("invalid flow"))
			continue
		}
		if !filter(flow) {
			continue
		}
		ev := flow.ToEvent(true)
		evs = append(evs, ev.BeatEvent("system", "socket"))
	}
	return evs, errs.Err()
}

func assertValue(t *testing.T, ev beat.Event, expected interface{}, field string) bool {
	value, err := ev.GetValue(field)
	return assert.Nil(t, err, field) && assert.Equal(t, expected, value, field)
}

func be16(val uint16) uint16 {
	var buf [2]byte
	binary.BigEndian.PutUint16(buf[:], val)
	return tracing.MachineEndian.Uint16(buf[:])
}

func be32(val uint32) uint32 {
	var buf [4]byte
	binary.BigEndian.PutUint32(buf[:], val)
	return tracing.MachineEndian.Uint32(buf[:])
}

func be64(val uint64) uint64 {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], val)
	return tracing.MachineEndian.Uint64(buf[:])
}

func all(*common.Flow) bool {
	return true
}
