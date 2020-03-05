// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,386 linux,amd64

package tests

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/sys/unix"

	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system/socket/dns"
	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system/socket/events"
	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system/socket/state"
	"github.com/elastic/beats/v7/x-pack/auditbeat/tracing"
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
	st := state.MakeState(nil, (*logWrapper)(t), time.Second, time.Second, 0, time.Second)
	lPort, rPort := be16(localPort), be16(remotePort)
	lAddr, rAddr := ipv4(localIP), ipv4(remoteIP)
	evs := []events.Event{
		execveCall(meta(1234, 1234, 1), []string{"/usr/bin/curl", "https://example.net/", "-o", "/tmp/site.html"}),
		commitCredsCall(meta(1234, 1234, 2), 501, 20, 501, 20),
		execveReturn(meta(1234, 1234, 2), 1234),
		inetCreateCall(meta(1234, 1235, 5), 0),
		sockInitDataCall(meta(1234, 1235, 5), sock),
		tcpv4ConnectCall(meta(1234, 1235, 8), sock, rAddr, rPort),
		ipLocalOutCall(meta(1234, 1235, 8), sock, 20, lAddr, rAddr, lPort, rPort),
		tcpConnectReturn(meta(1234, 1235, 9), 0),
		tcpv4DoRcvCall(meta(0, 0, 12), sock, 12, lAddr, rAddr, lPort, rPort),
		inetReleaseCall(meta(0, 0, 15), sock),
		tcpv4DoRcvCall(meta(0, 0, 17), sock, 7, lAddr, rAddr, lPort, rPort),
		doExitCall(meta(1234, 1234, 18)),
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

func TestTCPConnWithProcessSocketTimeouts(t *testing.T) {
	const (
		localIP            = "192.168.33.10"
		remoteIP           = "172.19.12.13"
		localPort          = 38842
		remotePort         = 443
		sock       uintptr = 0xff1234
	)
	st := state.MakeState(nil, (*logWrapper)(t), time.Second, 0, 0, time.Second)
	lPort, rPort := be16(localPort), be16(remotePort)
	lAddr, rAddr := ipv4(localIP), ipv4(remoteIP)
	evs := []events.Event{
		execveCall(meta(1234, 1234, 1), []string{"/usr/bin/curl", "https://example.net/", "-o", "/tmp/site.html"}),
		commitCredsCall(meta(1234, 1234, 2), 501, 20, 501, 20),
		execveReturn(meta(1234, 1234, 2), 1234),
		inetCreateCall(meta(1234, 1235, 5), 0),
		sockInitDataCall(meta(1234, 1235, 5), sock),
		tcpv4ConnectCall(meta(1234, 1235, 8), sock, rAddr, rPort),
		ipLocalOutCall(meta(1234, 1235, 8), sock, 20, lAddr, rAddr, lPort, rPort),
		tcpConnectReturn(meta(1234, 1235, 9), 0),
		tcpv4DoRcvCall(meta(0, 0, 12), sock, 12, lAddr, rAddr, lPort, rPort),
	}
	feedEvents(evs, st, t)
	st.CleanUp()
	evs = []events.Event{
		inetReleaseCall(meta(0, 0, 15), sock),
		tcpv4DoRcvCall(meta(0, 0, 17), sock, 7, lAddr, rAddr, lPort, rPort),
		doExitCall(meta(1234, 1234, 18)),
	}
	feedEvents(evs, st, t)
	st.CleanUp()
	flows, err := getFlows(st.PopFlows(), all)
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, flows, 2)
	flow := flows[0]
	t.Log("read flow 0", flow)
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

	// we have a truncated flow with no directionality,
	// so just report what we
	flow = flows[1]
	t.Log("read flow 1", flow)
	for field, expected := range map[string]interface{}{
		// swap the source and destination since we have a truncated flow
		// and don't know if the initial transaction was a dial out or dial in
		// but the first thing we got was a Rcv
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
	st := state.MakeState(nil, (*logWrapper)(t), time.Second, time.Second, 0, time.Second)
	lPort, rPort := be16(localPort), be16(remotePort)
	lAddr, rAddr := ipv4(localIP), ipv4(remoteIP)
	evs := []events.Event{
		execveCall(meta(1234, 1234, 1), []string{"/usr/bin/exfil-udp"}),
		commitCredsCall(meta(1234, 1234, 2), 501, 20, 501, 20),
		execveReturn(meta(1234, 1234, 2), 1234),
		inetCreateCall(meta(1234, 1235, 5), 0),
		sockInitDataCall(meta(1234, 1235, 5), sock),
		udpSendmsgCall(meta(1234, 1235, 6), sock, 123, 0, lAddr, 0, rAddr, lPort, 0, rPort, 0),
		inetReleaseCall(meta(1234, 12345, 15), sock),
		doExitCall(meta(1234, 1234, 18)),
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
	st := state.MakeState(nil, (*logWrapper)(t), time.Second, time.Second, 0, time.Second)
	lPort, rPort := be16(localPort), be16(remotePort)
	lAddr, rAddr := ipv4(localIP), ipv4(remoteIP)
	var packet [256]byte
	var ipHdr, udpHdr uint16 = 2, 64
	packet[ipHdr] = 0x45
	tracing.MachineEndian.PutUint32(packet[ipHdr+12:], rAddr)
	tracing.MachineEndian.PutUint16(packet[udpHdr:], rPort)
	evs := []events.Event{
		execveCall(meta(1234, 1234, 1), []string{"/usr/bin/exfil-udp"}),
		commitCredsCall(meta(1234, 1234, 2), 501, 20, 501, 20),
		execveReturn(meta(1234, 1234, 2), 1234),
		inetCreateCall(meta(1234, 1235, 5), 0),
		sockInitDataCall(meta(1234, 1235, 5), sock),
		udpQueueRcvSkbCall(meta(1234, 1235, 5), sock, 123, lAddr, lPort, ipHdr, udpHdr, packet),
		inetReleaseCall(meta(1234, 12345, 15), sock),
		doExitCall(meta(1234, 1234, 18)),
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
	proc       *state.Process
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
		proc1 := (&state.Process{}).SetPID(123)
		proc2 := (&state.Process{}).SetPID(124)
		tracker := state.NewDNSTracker(infiniteExpiration)
		tracker.AddTransaction(trV4)
		tracker.AddTransaction(trV6)
		tracker.RegisterEndpoint(local1, proc1)
		tracker.RegisterEndpoint(local2, proc1)
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
		proc1 := (&state.Process{}).SetPID(123)
		proc2 := (&state.Process{}).SetPID(124)
		tracker := state.NewDNSTracker(infiniteExpiration)
		tracker.RegisterEndpoint(local1, proc1)
		tracker.RegisterEndpoint(local2, proc1)
		tracker.AddTransaction(trV4)
		tracker.AddTransaction(trV6)
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
		proc1 := (&state.Process{}).SetPID(123)
		proc2 := (&state.Process{}).SetPID(124)
		tracker := state.NewDNSTracker(infiniteExpiration)
		tracker.RegisterEndpoint(local1, proc1)
		tracker.AddTransaction(trV4)
		tracker.AddTransaction(trV6)
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
		proc1 := (&state.Process{}).SetPID(123)
		tracker := state.NewDNSTracker(10 * time.Millisecond)
		tracker.AddTransaction(trV4)
		tracker.AddTransaction(trV6)
		time.Sleep(time.Millisecond * 50)
		tracker.RegisterEndpoint(local1, proc1)
		tracker.RegisterEndpoint(local2, proc1)
		dnsTestCases{
			{false, proc1, "192.0.2.12", ""},
			{false, proc1, "192.0.2.13", ""},
			{false, proc1, "2001:db8::1111", ""},
			{false, proc1, "2001:db8::2222", ""},
		}.Run(t)
	})
	t.Run("same IP different domains", func(t *testing.T) {
		proc1 := (&state.Process{}).SetPID(123)
		proc2 := (&state.Process{}).SetPID(124)
		trV4alt := dns.Transaction{
			TXID:      1234,
			Client:    local2,
			Server:    net.UDPAddr{IP: net.ParseIP("8.8.8.8"), Port: 53},
			Domain:    "example.com",
			Addresses: []net.IP{net.ParseIP("192.0.2.12"), net.ParseIP("192.0.2.13")},
		}
		tracker := state.NewDNSTracker(infiniteExpiration)
		tracker.AddTransaction(trV4)
		tracker.AddTransaction(trV4alt)
		tracker.RegisterEndpoint(local1, proc1)
		tracker.RegisterEndpoint(local2, proc2)
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
		ev := udpSendmsgCall(
			meta(1234, 1235, 6),
			0,
			0,
			0x7fffffff,
			ipv4("10.11.12.13"),
			ipv4("10.20.30.40"),
			ipv4("192.168.255.255"),
			be16(1010),
			be16(1234),
			be16(555),
			unix.AF_INET,
		)
		assert.Equal(t, expectedIPv4, ev.String())
	})
	t.Run("ipv4 connected", func(t *testing.T) {
		ev := udpSendmsgCall(
			meta(1234, 1235, 6),
			0,
			0,
			0,
			ipv4("10.11.12.13"),
			ipv4("192.168.255.255"),
			ipv4("10.20.30.40"),
			be16(1010),
			be16(555),
			be16(1234),
			0,
		)
		assert.Equal(t, expectedIPv4, ev.String())
	})
	t.Run("ipv6 non-connected", func(t *testing.T) {
		ev := udpv6SendmsgCall(
			meta(1234, 1235, 6),
			0x7fffffff,
			be16(1010),
			be16(1234),
			be16(555),
			unix.AF_INET6,
		)
		ev.LAddrA, ev.LAddrB = ipv6("fddd::bebe")
		ev.RAddrA, ev.RAddrB = ipv6("fddd::cafe")
		ev.AltRAddrA, ev.AltRAddrB = ipv6("fddd::bad:bad")
		assert.Equal(t, expectedIPv6, ev.String())
	})

	t.Run("ipv6 connected", func(t *testing.T) {
		ev := udpv6SendmsgCall(
			meta(1234, 1235, 6),
			0,
			be16(1010),
			be16(555),
			be16(1234),
			0,
		)
		ev.LAddrA, ev.LAddrB = ipv6("fddd::bebe")
		ev.RAddrA, ev.RAddrB = ipv6("fddd::bad:bad")
		ev.AltRAddrA, ev.AltRAddrB = ipv6("fddd::cafe")
		assert.Equal(t, expectedIPv6, ev.String())
	})

}
