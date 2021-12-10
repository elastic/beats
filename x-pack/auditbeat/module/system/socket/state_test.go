// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build (linux && 386) || (linux && amd64)
// +build linux,386 linux,amd64

package socket

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/joeshaw/multierror"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sys/unix"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system/socket/dns"
	"github.com/elastic/beats/v7/x-pack/auditbeat/tracing"
)

type logWrapper testing.T

func (l *logWrapper) Errorf(format string, args ...interface{}) {
	l.Logf("error: "+format, args...)
}

func (l *logWrapper) Warnf(format string, args ...interface{}) {
	l.Logf("warning: "+format, args...)
}

func (l *logWrapper) Infof(format string, args ...interface{}) {
	l.Logf("info: "+format, args...)
}

func (l *logWrapper) Debugf(format string, args ...interface{}) {
	l.Logf("debug: "+format, args...)
}

func TestTCPConnWithProcess(t *testing.T) {
	const (
		localIP            = "192.168.33.10"
		remoteIP           = "172.19.12.13"
		localPort          = 38842
		remotePort         = 443
		sock       uintptr = 0xff1234
	)
	st := makeState(nil, (*logWrapper)(t), time.Second, time.Second, 0, time.Second)
	lPort, rPort := be16(localPort), be16(remotePort)
	lAddr, rAddr := ipv4(localIP), ipv4(remoteIP)
	evs := []event{
		callExecve(meta(1234, 1234, 1), []string{"/usr/bin/curl", "https://example.net/", "-o", "/tmp/site.html"}),
		&commitCreds{Meta: meta(1234, 1234, 2), UID: 0, GID: 20, EUID: 501, EGID: 20},
		&execveRet{Meta: meta(1234, 1234, 2), Retval: 1234},
		&inetCreate{Meta: meta(1234, 1235, 5), Proto: 0},
		&sockInitData{Meta: meta(1234, 1235, 5), Sock: sock},
		&tcpIPv4ConnectCall{Meta: meta(1234, 1235, 8), Sock: sock, RAddr: rAddr, RPort: rPort},
		&ipLocalOutCall{
			Meta:  meta(1234, 1235, 8),
			Sock:  sock,
			Size:  20,
			LAddr: lAddr,
			LPort: lPort,
			RAddr: rAddr,
			RPort: rPort,
		},
		&tcpConnectResult{Meta: meta(1234, 1235, 9), Retval: 0},
		&tcpV4DoRcv{
			Meta:  meta(0, 0, 12),
			Sock:  sock,
			Size:  12,
			LAddr: lAddr,
			LPort: lPort,
			RAddr: rAddr,
			RPort: rPort,
		},
		&inetReleaseCall{Meta: meta(0, 0, 15), Sock: sock},
		&tcpV4DoRcv{
			Meta:  meta(0, 0, 17),
			Sock:  sock,
			Size:  7,
			LAddr: lAddr,
			LPort: lPort,
			RAddr: rAddr,
			RPort: rPort,
		},
		&doExit{Meta: meta(1234, 1234, 18)},
	}
	if err := feedEvents(evs, st, t); err != nil {
		t.Fatal(err)
	}
	st.ExpireOlder()
	flows, err := getFlows(st.DoneFlows(), all)
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
		"network.direction":   "egress",
		"network.transport":   "tcp",
		"network.type":        "ipv4",
		"process.pid":         1234,
		"process.name":        "curl",
		"user.id":             "0",
		"user.name":           "root",
		"event.type":          []string{"info", "connection"},
		"event.category":      []string{"network", "network_traffic"},
		"related.ip":          []string{localIP, remoteIP},
		"related.user":        []string{"root"},
	} {
		if !assertValue(t, flow, expected, field) {
			t.Fatal("expected value not found")
		}
	}
}

func TestTCPConnWithProcessSocketTimeouts(t *testing.T) {
	const (
		localIP               = "192.168.33.10"
		remoteIP              = "172.19.12.13"
		localPort             = 38842
		remotePort            = 443
		sock          uintptr = 0xff1234
		flowTimeout           = time.Hour
		socketTimeout         = time.Minute * 3
		closeTimeout          = time.Minute
	)
	st := makeState(nil, (*logWrapper)(t), flowTimeout, socketTimeout, closeTimeout, time.Second)
	now := time.Now()
	st.clock = func() time.Time {
		return now
	}
	lPort, rPort := be16(localPort), be16(remotePort)
	lAddr, rAddr := ipv4(localIP), ipv4(remoteIP)
	evs := []event{
		callExecve(meta(1234, 1234, 1), []string{"/usr/bin/curl", "https://example.net/", "-o", "/tmp/site.html"}),
		&commitCreds{Meta: meta(1234, 1234, 2), UID: 501, GID: 20, EUID: 501, EGID: 20},
		&execveRet{Meta: meta(1234, 1234, 2), Retval: 1234},
		&inetCreate{Meta: meta(1234, 1235, 5), Proto: 0},
		&sockInitData{Meta: meta(1234, 1235, 5), Sock: sock},
		&tcpIPv4ConnectCall{Meta: meta(1234, 1235, 8), Sock: sock, RAddr: rAddr, RPort: rPort},
		&ipLocalOutCall{
			Meta:  meta(1234, 1235, 8),
			Sock:  sock,
			Size:  20,
			LAddr: lAddr,
			LPort: lPort,
			RAddr: rAddr,
			RPort: rPort,
		},
		&tcpConnectResult{Meta: meta(1234, 1235, 9), Retval: 0},
		&tcpV4DoRcv{
			Meta:  meta(0, 0, 12),
			Sock:  sock,
			Size:  12,
			LAddr: lAddr,
			LPort: lPort,
			RAddr: rAddr,
			RPort: rPort,
		},
	}
	if err := feedEvents(evs, st, t); err != nil {
		t.Fatal(err)
	}
	st.ExpireOlder()
	// Nothing expired just yet.
	flows, err := getFlows(st.DoneFlows(), all)
	if err != nil {
		t.Fatal(err)
	}
	assert.Empty(t, flows)

	evs = []event{
		&clockSyncCall{
			Meta: meta(uint32(os.Getpid()), 1235, 0),
			Ts:   uint64(now.UnixNano()),
		},
		&inetReleaseCall{Meta: meta(0, 0, 15), Sock: sock},
		&tcpV4DoRcv{
			Meta:  meta(0, 0, 17),
			Sock:  sock,
			Size:  7,
			LAddr: lAddr,
			LPort: lPort,
			RAddr: rAddr,
			RPort: rPort,
		},

		&inetCreate{Meta: meta(1234, 1235, 18), Proto: 0},
		&sockInitData{Meta: meta(1234, 1235, 19), Sock: sock + 1},
		&tcpIPv4ConnectCall{Meta: meta(1234, 1235, 20), Sock: sock + 1, RAddr: rAddr, RPort: rPort},
		&tcpV4DoRcv{
			Meta:  meta(0, 0, 21),
			Sock:  sock + 1,
			Size:  12,
			LAddr: lAddr,
			LPort: lPort,
			RAddr: rAddr,
			RPort: rPort,
		},
	}
	if err := feedEvents(evs, st, t); err != nil {
		t.Fatal(err)
	}
	// Expire the first socket
	now = now.Add(closeTimeout + 1)
	st.ExpireOlder()
	flows, err = getFlows(st.DoneFlows(), all)
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, flows, 1)
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
		"destination.packets": uint64(2),
		"destination.bytes":   uint64(19),
		"server.ip":           remoteIP,
		"server.port":         remotePort,
		"network.direction":   "egress",
		"network.transport":   "tcp",
		"network.type":        "ipv4",
		"process.pid":         1234,
		"process.name":        "curl",
		"user.id":             "501",
		"event.type":          []string{"info", "connection"},
		"event.category":      []string{"network", "network_traffic"},
	} {
		if !assertValue(t, flow, expected, field) {
			t.Fatal("expected value not found")
		}
	}
	// Wait until sock+1 expires due to inactivity. It won't be available
	// just yet.
	now = now.Add(socketTimeout + 1)
	st.ExpireOlder()
	flows, err = getFlows(st.DoneFlows(), all)
	if err != nil {
		t.Fatal(err)
	}
	assert.Empty(t, flows)

	// Wait until the sock is closed completely.
	now = now.Add(closeTimeout + 1)
	st.ExpireOlder()
	flows, err = getFlows(st.DoneFlows(), all)
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, flows, 1)
	flow = flows[0]

	// we have a truncated flow with no directionality,
	// so just report what we can
	t.Log("read flow 1", flow)
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
		"event.type":        []string{"info", "connection"},
		"event.category":    []string{"network", "network_traffic"},
	} {
		if !assertValue(t, flow, expected, field) {
			t.Fatal("expected value not found")
		}
	}
}

func TestSocketExpirationWithOverwrittenSockets(t *testing.T) {
	const (
		sock          uintptr = 0xff1234
		flowTimeout           = time.Hour
		socketTimeout         = time.Minute * 3
		closeTimeout          = time.Minute
	)
	st := makeState(nil, (*logWrapper)(t), flowTimeout, socketTimeout, closeTimeout, time.Second)
	now := time.Now()
	st.clock = func() time.Time {
		return now
	}
	if err := feedEvents([]event{
		&inetCreate{Meta: meta(1234, 1236, 5), Proto: 0},
		&sockInitData{Meta: meta(1234, 1236, 5), Sock: sock},
		&inetCreate{Meta: meta(1234, 1237, 5), Proto: 0},
		&sockInitData{Meta: meta(1234, 1237, 5), Sock: sock},
	}, st, t); err != nil {
		t.Fatal(err)
	}
	now = now.Add(closeTimeout + 1)
	st.ExpireOlder()
	now = now.Add(socketTimeout + 1)
	st.ExpireOlder()
}

func TestUDPOutgoingSinglePacketWithProcess(t *testing.T) {
	const (
		localIP            = "192.168.33.10"
		remoteIP           = "172.19.12.13"
		localPort          = 38842
		remotePort         = 53
		sock       uintptr = 0xff1234
	)
	st := makeState(nil, (*logWrapper)(t), time.Second, time.Second, 0, time.Second)
	lPort, rPort := be16(localPort), be16(remotePort)
	lAddr, rAddr := ipv4(localIP), ipv4(remoteIP)
	evs := []event{
		callExecve(meta(1234, 1234, 1), []string{"/usr/bin/exfil-udp"}),
		&commitCreds{Meta: meta(1234, 1234, 2), UID: 501, GID: 20, EUID: 501, EGID: 20},
		&execveRet{Meta: meta(1234, 1234, 2), Retval: 1234},
		&inetCreate{Meta: meta(1234, 1235, 5), Proto: 0},
		&sockInitData{Meta: meta(1234, 1235, 5), Sock: sock},
		&udpSendMsgCall{
			Meta:     meta(1234, 1235, 6),
			Sock:     sock,
			Size:     123,
			LAddr:    lAddr,
			AltRAddr: rAddr,
			LPort:    lPort,
			AltRPort: rPort,
		},
		&inetReleaseCall{Meta: meta(1234, 1235, 17), Sock: sock},
		&doExit{Meta: meta(1234, 1234, 18)},
	}
	if err := feedEvents(evs, st, t); err != nil {
		t.Fatal(err)
	}
	st.ExpireOlder()
	flows, err := getFlows(st.DoneFlows(), all)
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
		"network.direction":   "egress",
		"network.transport":   "udp",
		"network.type":        "ipv4",
		"process.pid":         1234,
		"process.name":        "exfil-udp",
		"user.id":             "501",
		"event.type":          []string{"info", "connection"},
		"event.category":      []string{"network", "network_traffic"},
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
	st := makeState(nil, (*logWrapper)(t), time.Second, time.Second, 0, time.Second)
	lPort, rPort := be16(localPort), be16(remotePort)
	lAddr, rAddr := ipv4(localIP), ipv4(remoteIP)
	var packet [256]byte
	var ipHdr, udpHdr uint16 = 2, 64
	packet[ipHdr] = 0x45
	tracing.MachineEndian.PutUint32(packet[ipHdr+12:], rAddr)
	tracing.MachineEndian.PutUint16(packet[udpHdr:], rPort)
	evs := []event{
		callExecve(meta(1234, 1234, 1), []string{"/usr/bin/exfil-udp"}),
		&commitCreds{Meta: meta(1234, 1234, 2), UID: 501, GID: 20, EUID: 501, EGID: 20},
		&execveRet{Meta: meta(1234, 1234, 2), Retval: 1234},
		&inetCreate{Meta: meta(1234, 1235, 5), Proto: 0},
		&sockInitData{Meta: meta(1234, 1235, 5), Sock: sock},
		&udpQueueRcvSkb{
			Meta:   meta(1234, 1235, 5),
			Sock:   sock,
			Size:   123,
			LAddr:  lAddr,
			LPort:  lPort,
			IPHdr:  ipHdr,
			UDPHdr: udpHdr,
			Packet: packet,
		},
		&inetReleaseCall{Meta: meta(1234, 1235, 17), Sock: sock},
		&doExit{Meta: meta(1234, 1234, 18)},
	}
	if err := feedEvents(evs, st, t); err != nil {
		t.Fatal(err)
	}
	st.ExpireOlder()
	flows, err := getFlows(st.DoneFlows(), all)
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
		"network.direction":   "ingress",
		"network.transport":   "udp",
		"network.type":        "ipv4",
		"process.pid":         1234,
		"process.name":        "exfil-udp",
		"user.id":             "501",
		"event.type":          []string{"info", "connection"},
		"event.category":      []string{"network", "network_traffic"},
	} {
		assertValue(t, flow, expected, field)
	}
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

func feedEvents(evs []event, st *state, t *testing.T) error {
	for idx, ev := range evs {
		t.Logf("Delivering event %d: %s", idx, ev.String())
		// TODO: err
		if err := ev.Update(st); err != nil {
			return errors.Wrapf(err, "error feeding event '%s'", ev.String())
		}
	}
	return nil
}

func all(*flow) bool {
	return true
}

type noDNSResolution struct{}

func (noDNSResolution) ResolveIP(pid uint32, ip net.IP) (domain string, found bool) {
	return "", false
}

func getFlows(list linkedList, filter func(*flow) bool) (evs []beat.Event, err error) {
	var errs multierror.Errors
	for elem := list.get(); elem != nil; elem = list.get() {
		flow, ok := elem.(*flow)
		if !ok || !flow.isValid() {
			errs = append(errs, errors.New("invalid flow"))
			continue
		}
		if !filter(flow) {
			continue
		}
		ev, err := flow.toEvent(true)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		evs = append(evs, ev.BeatEvent(moduleName, metricsetName))
	}
	return evs, errs.Err()
}

func callExecve(meta tracing.Metadata, args []string) *execveCall {
	ptr := &execveCall{
		Meta: meta,
	}
	lim := len(args)
	if lim > maxProgArgs {
		lim = maxProgArgs
	}
	for i := 0; i < lim; i++ {
		ptr.Ptrs[i] = 1
	}
	if lim < len(args) {
		ptr.Ptrs[lim] = 1
	}
	switch lim {
	case 5:
		copyCString(ptr.Param4[:], []byte(args[4]))
		fallthrough
	case 4:
		copyCString(ptr.Param3[:], []byte(args[3]))
		fallthrough
	case 3:
		copyCString(ptr.Param2[:], []byte(args[2]))
		fallthrough
	case 2:
		copyCString(ptr.Param1[:], []byte(args[1]))
		fallthrough
	case 1:
		copyCString(ptr.Param0[:], []byte(args[0]))
	case 0:
		return nil
	}
	ptr.Path = ptr.Param0
	return ptr
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

type dnsTestCase struct {
	found      bool
	proc       *process
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
		proc1 := &process{pid: 123}
		proc2 := &process{pid: 124}
		tracker := newDNSTracker(infiniteExpiration)
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
		proc1 := &process{pid: 123}
		proc2 := &process{pid: 124}
		tracker := newDNSTracker(infiniteExpiration)
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
		proc1 := &process{pid: 123}
		proc2 := &process{pid: 124}
		tracker := newDNSTracker(infiniteExpiration)
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
		proc1 := &process{pid: 123}
		tracker := newDNSTracker(10 * time.Millisecond)
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
		proc1 := &process{pid: 123}
		proc2 := &process{pid: 124}
		trV4alt := dns.Transaction{
			TXID:      1234,
			Client:    local2,
			Server:    net.UDPAddr{IP: net.ParseIP("8.8.8.8"), Port: 53},
			Domain:    "example.com",
			Addresses: []net.IP{net.ParseIP("192.0.2.12"), net.ParseIP("192.0.2.13")},
		}
		tracker := newDNSTracker(infiniteExpiration)
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
		ev := udpSendMsgCall{
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
		ev := udpSendMsgCall{
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
		ev := udpv6SendMsgCall{
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
		ev := udpv6SendMsgCall{
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

func TestSocketReuse(t *testing.T) {
	const (
		localIP            = "192.168.33.10"
		remoteIP           = "172.19.12.13"
		localPort          = 38842
		remotePort         = 53
		sock       uintptr = 0xff1234
	)
	st := makeState(nil, (*logWrapper)(t), time.Hour, time.Hour, 0, time.Hour)
	lPort, rPort := be16(localPort), be16(remotePort)
	lAddr, rAddr := ipv4(localIP), ipv4(remoteIP)
	evs := []event{
		&clockSyncCall{
			Meta: meta(uint32(os.Getpid()), 1235, 5),
			Ts:   uint64(time.Now().UnixNano()),
		},
		&inetCreate{Meta: meta(1234, 1235, 5), Proto: 0},
		&sockInitData{Meta: meta(1234, 1235, 5), Sock: sock},
		&udpSendMsgCall{
			Meta:     meta(1234, 1235, 6),
			Sock:     sock,
			Size:     123,
			LAddr:    lAddr,
			AltRAddr: rAddr,
			LPort:    lPort,
			AltRPort: rPort,
		},
		// Asume inetRelease lost.
		&inetCreate{Meta: meta(1234, 1235, 5), Proto: 0},
		&sockInitData{Meta: meta(1234, 1235, 5), Sock: sock},
		&udpSendMsgCall{
			Meta:     meta(1234, 1235, 6),
			Sock:     sock,
			Size:     123,
			LAddr:    lAddr,
			AltRAddr: rAddr,
			LPort:    lPort,
			AltRPort: rPort,
		},
	}
	if err := feedEvents(evs, st, t); err != nil {
		t.Fatal(err)
	}
	st.ExpireOlder()
	flows, err := getFlows(st.DoneFlows(), all)
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, flows, 1)
}

func TestProcessDNSRace(t *testing.T) {
	p := new(process)
	var wg sync.WaitGroup
	wg.Add(2)
	address := func(i byte) net.IP { return net.IPv4(172, 16, 0, i) }
	go func() {
		for i := byte(255); i > 0; i-- {
			p.addTransaction(dns.Transaction{
				Client:    net.UDPAddr{IP: net.IPv4(10, 20, 30, 40)},
				Server:    net.UDPAddr{IP: net.IPv4(10, 20, 30, 41)},
				Domain:    "example.net",
				Addresses: []net.IP{address(i)},
			})
		}
		wg.Done()
	}()
	go func() {
		for i := byte(255); i > 0; i-- {
			p.ResolveIP(address(i))
		}
		wg.Done()
	}()
	wg.Wait()
}
