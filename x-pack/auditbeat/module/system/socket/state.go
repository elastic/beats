// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// +build linux,386 linux,amd64

package socket

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/sys/unix"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/flowhash"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/x-pack/auditbeat/module/system/socket/dns"
	"github.com/elastic/beats/x-pack/auditbeat/module/system/socket/helper"
	"github.com/elastic/beats/x-pack/auditbeat/tracing"
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

type kernelTime uint64

type flowProto uint8

const (
	protoUnknown flowProto = 0
	protoTCP     flowProto = unix.IPPROTO_TCP
	protoUDP     flowProto = unix.IPPROTO_UDP
)

func (p flowProto) String() string {
	switch p {
	case protoTCP:
		return "tcp"
	case protoUDP:
		return "udp"
	}
	return "unknown"
}

type inetType uint8

const (
	inetTypeUnknown inetType = 0
	inetTypeIPv4    inetType = unix.AF_INET
	inetTypeIPv6    inetType = unix.AF_INET6
)

func (t inetType) String() string {
	switch t {
	case inetTypeIPv4:
		return "ipv4"
	case inetTypeIPv6:
		return "ipv6"
	}
	return "unknown"
}

type flowDirection uint8

const (
	directionUnknown flowDirection = iota
	directionInbound
	directionOutbound
)

// String returns the textual representation of the flowDirection.
func (d flowDirection) String() string {
	switch d {
	case directionInbound:
		return "inbound"
	case directionOutbound:
		return "outbound"
	default:
		return "unknown"
	}
}

type endpoint struct {
	addr           net.TCPAddr
	packets, bytes uint64
}

func (e *endpoint) updateWith(other endpoint) {
	if e.addr.IP == nil {
		e.addr.IP = other.addr.IP
		e.addr.Port = other.addr.Port
	}
	e.packets += other.packets
	e.bytes += other.bytes
}

// String returns the textual representation of the endpoint address:port.
func (e *endpoint) String() string {
	if e.addr.IP != nil {
		return e.addr.String()
	}
	return "(not bound)"
}

func newEndpointIPv4(beIP uint32, bePort uint16, pkts uint64, bytes uint64) (e endpoint) {
	var buf [4]byte
	e.packets = pkts
	e.bytes = bytes
	if bePort != 0 && beIP != 0 {
		tracing.MachineEndian.PutUint16(buf[:], bePort)
		port := binary.BigEndian.Uint16(buf[:])
		tracing.MachineEndian.PutUint32(buf[:], beIP)
		e.addr = net.TCPAddr{
			IP:   net.IPv4(buf[0], buf[1], buf[2], buf[3]),
			Port: int(port),
		}
	}
	return e
}

func newEndpointIPv6(beIPa uint64, beIPb uint64, bePort uint16, pkts uint64, bytes uint64) (e endpoint) {
	e.packets = pkts
	e.bytes = bytes
	if bePort != 0 && (beIPa != 0 || beIPb != 0) {
		addr := make([]byte, 16)
		tracing.MachineEndian.PutUint16(addr[:], bePort)
		port := binary.BigEndian.Uint16(addr[:])
		tracing.MachineEndian.PutUint64(addr, beIPa)
		tracing.MachineEndian.PutUint64(addr[8:], beIPb)
		e.addr = net.TCPAddr{
			IP:   addr,
			Port: int(port),
		}
	}
	return e
}

type linkedElement interface {
	Prev() linkedElement
	Next() linkedElement
	SetPrev(linkedElement)
	SetNext(linkedElement)
	Timestamp() time.Time
}

type linkedList struct {
	head, tail linkedElement
	size       uint
}

type flow struct {
	prev, next linkedElement

	sock              uintptr
	inetType          inetType
	proto             flowProto
	dir               flowDirection
	created, lastSeen kernelTime
	pid               uint32
	process           *process
	local, remote     endpoint
	complete          bool

	// these are automatically calculated by state from kernelTimes above
	createdTime, lastSeenTime time.Time
}

// If this flow should be reported or only captured partial data
func (f *flow) isValid() bool {
	return f.inetType != inetTypeUnknown && f.proto != protoUnknown && f.local.addr.IP != nil && f.remote.addr.IP != nil
}

// Prev returns the previous flow in a linked list of flows.
func (f *flow) Prev() linkedElement {
	return f.prev
}

// Next returns the next flow in a linked list of flows.
func (f *flow) Next() linkedElement {
	return f.next
}

// SetPrev sets previous flow in a linked list of flows.
func (f *flow) SetPrev(e linkedElement) {
	f.prev = e
}

// SetNext sets the next flow in a linked list of flows.
func (f *flow) SetNext(e linkedElement) {
	f.next = e
}

// Timestamp returns the time value used to expire this flow.
func (f *flow) Timestamp() time.Time {
	return f.lastSeenTime
}

type process struct {
	pid                  uint32
	name, path           string
	args                 []string
	created              kernelTime
	uid, gid, euid, egid uint32
	hasCreds             bool

	// populated by state from created
	createdTime time.Time
}

type socket struct {
	sock  uintptr
	flows map[string]*flow
	// Sockets have direction if they have been connect()ed or accept()ed.
	dir     flowDirection
	bound   bool
	pid     uint32
	process *process
	// This signals that the socket is in the closeTimeout list.
	closing    bool
	closeTime  time.Time
	prev, next linkedElement
}

// Prev returns the previous socket in the linked list.
func (s *socket) Prev() linkedElement {
	return s.prev
}

// Next returns the next socket in the linked list.
func (s *socket) Next() linkedElement {
	return s.next
}

// SetPrev sets the previous socket in the linked list.
func (s *socket) SetPrev(e linkedElement) {
	s.prev = e
}

// SetNext sets the next socket in the linked list.
func (s *socket) SetNext(e linkedElement) {
	s.next = e
}

// Timestamp returns the time reference used to expire sockets.
func (s *socket) Timestamp() time.Time {
	return s.closeTime
}

type pidAddress struct {
	pid  uint32
	addr net.IP
}

// String returns the string representation.
func (p pidAddress) String() string {
	return strconv.Itoa(int(p.pid)) + "|" + p.addr.String()
}

type dnsResolver interface {
	// ResolveIP returns the domain associated to the given IP.
	ResolveIP(pid uint32, ip net.IP) (domain string, found bool)
}

type dnsTracker struct {
	sync.Mutex
	// map[net.UDPAddr(string)][]dns.Transaction
	transactionByClient *common.Cache

	// map[net.UDPAddr(string)]uint32
	pidByClient *common.Cache

	// map[pidAddress(string)]string
	reverseHosts *common.Cache
}

func newDNSTracker(timeout time.Duration) dnsTracker {
	return dnsTracker{
		transactionByClient: common.NewCache(timeout, 8),
		pidByClient:         common.NewCache(timeout, 8),
		reverseHosts:        common.NewCache(timeout, 8),
	}
}

// AddTransaction registers a new DNS transaction.
func (dt *dnsTracker) AddTransaction(tr dns.Transaction) {
	dt.Lock()
	defer dt.Unlock()
	clientAddr := tr.Client.String()
	if pidIf := dt.pidByClient.Get(clientAddr); pidIf != nil {
		if pid, ok := pidIf.(uint32); ok {
			dt.addTransactionWithPID(tr, pid)
			return
		}
	}
	var list []dns.Transaction
	if prev := dt.transactionByClient.Get(clientAddr); prev != nil {
		list = prev.([]dns.Transaction)
	}
	list = append(list, tr)
	dt.transactionByClient.Put(clientAddr, list)
}

func (dt *dnsTracker) addTransactionWithPID(tr dns.Transaction, pid uint32) {
	for _, addr := range tr.Addresses {
		dt.reverseHosts.Put(pidAddress{pid: pid, addr: addr}.String(), tr.Domain)
	}
}

// AddTransactionWithPID registers a new DNS transaction from the given PID.
func (dt *dnsTracker) AddTransactionWithPID(tr dns.Transaction, pid uint32) {
	dt.Lock()
	defer dt.Unlock()
	dt.addTransactionWithPID(tr, pid)
}

// CleanUp removes expired entries from the maps.
func (dt *dnsTracker) CleanUp() {
	dt.Lock()
	defer dt.Unlock()
	dt.transactionByClient.CleanUp()
	dt.pidByClient.CleanUp()
	dt.reverseHosts.CleanUp()
}

// RegisterEndpoint registers a new local endpoint used for DNS queries
// to correlate captured DNS packets with their originator process.
func (dt *dnsTracker) RegisterEndpoint(addr net.UDPAddr, pid uint32) {
	dt.Lock()
	defer dt.Unlock()
	key := addr.String()
	dt.pidByClient.Put(key, pid)
	if listIf := dt.transactionByClient.Get(key); listIf != nil {
		list := listIf.([]dns.Transaction)
		for _, tr := range list {
			dt.addTransactionWithPID(tr, pid)
		}
	}
}

// ResolveIP returns the domain associated with the given IP.
func (dt *dnsTracker) ResolveIP(pid uint32, ip net.IP) (domain string, found bool) {
	dt.Lock()
	defer dt.Unlock()
	if domainIf := dt.reverseHosts.Get(pidAddress{pid: pid, addr: ip}.String()); domainIf != nil {
		domain, found = domainIf.(string)
	}
	return
}

type state struct {
	sync.Mutex
	// Used to convert kernel time to user time
	kernelEpoch time.Time

	reporter mb.PushReporterV2
	log      helper.Logger

	processes map[uint32]*process
	socks     map[uintptr]*socket
	threads   map[uint32]event

	numFlows uint64

	// configuration
	inactiveTimeout, closeTimeout time.Duration
	clockMaxDrift                 time.Duration

	// lru used for flow expiration.
	lru linkedList

	// holds closed and expired flows.
	done linkedList

	// holds sockets in closing state. This is to keep them around until their
	// close timeout expires.
	closing linkedList

	dns dnsTracker
}

func (s *state) getSocket(sock uintptr) *socket {
	if socket, found := s.socks[sock]; found {
		return socket
	}
	socket := &socket{
		sock: sock,
	}
	s.socks[sock] = socket
	return socket
}

var kernelProcess = process{
	pid:  0,
	name: "[kernel_task]",
}

func NewState(r mb.PushReporterV2, log helper.Logger, inactiveTimeout, closeTimeout, clockMaxDrift time.Duration) *state {
	s := makeState(r, log, inactiveTimeout, closeTimeout, clockMaxDrift)
	go s.reapLoop()
	return s
}

func makeState(r mb.PushReporterV2, log helper.Logger, inactiveTimeout, closeTimeout, clockMaxDrift time.Duration) *state {
	return &state{
		reporter:        r,
		log:             log,
		processes:       make(map[uint32]*process),
		socks:           make(map[uintptr]*socket),
		threads:         make(map[uint32]event),
		inactiveTimeout: inactiveTimeout,
		closeTimeout:    closeTimeout,
		clockMaxDrift:   clockMaxDrift,
		dns:             newDNSTracker(inactiveTimeout * 2),
	}
}

func (s *state) DoneFlows() linkedList {
	s.Lock()
	defer s.Unlock()
	r := s.done
	s.done = linkedList{}
	return r
}

var lastEvents uint64
var lastTime time.Time

func (s *state) logState() {
	s.Lock()
	numFlows := s.numFlows
	numSocks := len(s.socks)
	numProcs := len(s.processes)
	numThreads := len(s.threads)
	lruSize := s.lru.size
	doneSize := s.done.size
	closingSize := s.closing.size
	events := atomic.LoadUint64(&eventCount)
	s.Unlock()

	now := time.Now()
	took := now.Sub(lastTime)
	newEvs := events - lastEvents
	lastEvents = events
	lastTime = now
	var errs []string
	if uint64(lruSize) != numFlows {
		errs = append(errs, "flow count mismatch")
	}
	msg := fmt.Sprintf("state flows=%d sockets=%d procs=%d threads=%d lru=%d done=%d closing=%d events=%d eps=%.1f",
		numFlows, numSocks, numProcs, numThreads, lruSize, doneSize, closingSize, events,
		float64(newEvs)*float64(time.Second)/float64(took))
	if errs == nil {
		s.log.Debugf("%s", msg)
	} else {
		s.log.Warnf("%s. Warnings: %v", msg, errs)
	}

}

func (s *state) reapLoop() {
	reportTicker := time.NewTicker(reapInterval)
	defer reportTicker.Stop()
	logTicker := time.NewTicker(logInterval)
	defer logTicker.Stop()
	for {
		select {
		case <-s.reporter.Done():
			return
		case <-reportTicker.C:
			s.ExpireOlder()
			flows := s.DoneFlows()
			for elem := flows.get(); elem != nil; elem = flows.get() {
				flow, ok := elem.(*flow)
				if !ok || !flow.isValid() {
					continue
				}
				if int(flow.pid) == os.Getpid() {
					// Do not report flows for which we are the source
					// to prevent a feedback loop.
					continue
				}
				ev, err := flow.toEvent(true, &s.dns)
				if err != nil {
					s.log.Errorf("Failed to convert flow=%v err=%v", flow, err)
					continue
				}
				if !s.reporter.Event(ev) {
					return
				}
			}
		case <-logTicker.C:
			s.logState()
		}
	}
}

func (s *state) ExpireOlder() {
	s.Lock()
	defer s.Unlock()
	deadline := time.Now().Add(-s.inactiveTimeout)
	for item := s.lru.peek(); item != nil && item.Timestamp().Before(deadline); {
		if flow, ok := item.(*flow); ok {
			s.onFlowTerminated(flow)
		} else {
			s.lru.get()
		}
		item = s.lru.peek()
	}

	deadline = time.Now().Add(-s.closeTimeout)
	for item := s.closing.peek(); item != nil && item.Timestamp().Before(deadline); {
		if sock, ok := item.(*socket); ok {
			s.onSockTerminated(sock)
		} else {
			s.closing.get()
		}
		item = s.closing.peek()
	}
	// Expire cached DNS
	s.dns.CleanUp()
}

func (s *state) CreateProcess(p process) error {
	if p.pid == 0 {
		return errors.New("can't create process with PID 0")
	}
	s.Lock()
	defer s.Unlock()
	s.processes[p.pid] = &p
	if p.createdTime == (time.Time{}) {
		p.createdTime = s.kernTimestampToTime(p.created)
	}
	return nil
}

func (s *state) TerminateProcess(pid uint32) error {
	if pid == 0 {
		return errors.New("can't terminate process with PID 0")
	}
	s.Lock()
	defer s.Unlock()
	delete(s.processes, pid)
	return nil
}

func (s *state) getProcess(pid uint32) *process {
	if pid == 0 {
		return &kernelProcess
	}
	return s.processes[pid]
}

func (s *state) ThreadEnter(tid uint32, ev event) error {
	s.Lock()
	prev, hasPrev := s.threads[tid]
	s.threads[tid] = ev
	s.Unlock()
	if hasPrev {
		return fmt.Errorf("thread already had an event. tid=%d existing=%v", tid, prev)
	}
	return nil
}

func (s *state) ThreadLeave(tid uint32) (ev event, found bool) {
	s.Lock()
	defer s.Unlock()
	if ev, found = s.threads[tid]; found {
		delete(s.threads, tid)
	}
	return
}

func (s *state) onSockTerminated(sock *socket) {
	for _, flow := range sock.flows {
		s.onFlowTerminated(flow)
	}
	delete(s.socks, sock.sock)
	if sock.closing {
		s.closing.remove(sock)
	}
}

// CreateSocket allocates a new sock in the system
func (s *state) CreateSocket(ref flow) error {
	s.Lock()
	defer s.Unlock()
	ref.createdTime = s.kernTimestampToTime(ref.created)
	ref.lastSeenTime = s.kernTimestampToTime(ref.lastSeen)
	// terminate existing if sock ptr is reused
	if prev, found := s.socks[ref.sock]; found {
		s.onSockTerminated(prev)
	}
	return s.createFlow(ref)
}

func (s *state) OnDNSTransaction(tr dns.Transaction) error {
	s.Lock()
	defer s.Unlock()
	s.dns.AddTransaction(tr)
	return nil
}

func (s *state) mutualEnrich(sock *socket, f *flow) {
	// if the sock is not bound to a local address yet, update if possible
	if !sock.bound && f.local.addr.IP != nil {
		sock.bound = true
		for _, flow := range sock.flows {
			if flow.local.addr.IP == nil {
				flow.local.addr = f.local.addr
			}
		}
		if f.proto == protoUDP && f.remote.addr.Port == 53 {
			localUDP := net.UDPAddr{
				IP:   f.local.addr.IP,
				Port: f.local.addr.Port,
			}
			s.dns.RegisterEndpoint(localUDP, sock.pid)
		}
	}
	if sockNoDir := sock.dir == directionUnknown; sockNoDir != (f.dir == directionUnknown) {
		if sockNoDir {
			sock.dir = f.dir
		} else {
			f.dir = sock.dir
		}
	}
	if sockNoPID := sock.pid == 0; sockNoPID != (f.pid == 0) {
		if sockNoPID {
			sock.pid = f.pid
			sock.process = f.process
		} else {
			f.pid = sock.pid
			f.process = sock.process
		}
	}
}

func (s *state) createFlow(ref flow) error {
	// Get or create a socket for this flow
	sock, found := s.socks[ref.sock]
	if !found {
		sock = &socket{
			sock: ref.sock,
		}
		s.socks[ref.sock] = sock

	}

	ref.createdTime = ref.lastSeenTime
	s.mutualEnrich(sock, &ref)
	// don't create the flow yet if it doesn't have a populated remote address
	if ref.remote.addr.IP == nil {
		return nil
	}
	ptr := new(flow)
	*ptr = ref
	if sock.flows == nil {
		sock.flows = make(map[string]*flow, 1)
	}
	sock.flows[ref.remote.addr.String()] = ptr
	s.lru.add(ptr)
	s.numFlows++
	return nil
}

// OnSockDestroyed is called to signal that the given sock has been destroyed.
func (s *state) OnSockDestroyed(ptr uintptr, pid uint32) error {
	s.Lock()
	defer s.Unlock()
	sock, found := s.socks[ptr]
	if !found {
		return nil
	}
	// Enrich with pid
	if sock.pid == 0 && pid != 0 {
		sock.pid = pid
		sock.process = s.getProcess(pid)
	}
	// Keep the sock around in case it's a connected TCP socket, as still some
	// packets can be received shortly after/during inet_release.
	if !sock.closing {
		sock.closeTime = time.Now()
		sock.closing = true
		s.closing.add(sock)
	}
	return nil
}

// UpdateFlow receives a partial flow and creates or updates an existing flow.
func (s *state) UpdateFlow(ref flow) error {
	return s.UpdateFlowWithCondition(ref, nil)
}

// UpdateFlowWithCondition receives a partial flow and creates or updates an
// existing flow. The optional condition must be met before an existing flow is
// updated. Otherwise the update is ignored.
func (s *state) UpdateFlowWithCondition(ref flow, cond func(*flow) bool) error {
	s.Lock()
	defer s.Unlock()
	ref.createdTime = s.kernTimestampToTime(ref.created)
	ref.lastSeenTime = s.kernTimestampToTime(ref.lastSeen)
	sock, found := s.socks[ref.sock]
	if !found {
		return s.createFlow(ref)
	}
	prev, found := sock.flows[ref.remote.addr.String()]
	if !found {
		return s.createFlow(ref)
	}
	if cond != nil && !cond(prev) {
		return nil
	}
	s.mutualEnrich(sock, &ref)
	prev.updateWith(ref, s)
	s.lru.remove(prev)
	s.lru.add(prev)
	return nil
}

func (f *flow) updateWith(ref flow, s *state) {
	f.lastSeenTime = ref.lastSeenTime
	if ref.inetType != f.inetType {
		if f.inetType == inetTypeUnknown {
			f.inetType = ref.inetType
		}
	}
	if ref.proto != f.proto {
		if f.proto == protoUnknown {
			f.proto = ref.proto
		}
	}
	if f.pid == 0 && ref.pid != 0 {
		f.pid = ref.pid
		f.process = ref.process
	}
	if f.process == nil {
		if ref.process != nil && f.pid == ref.pid {
			f.process = ref.process
		} else {
			f.process = s.getProcess(f.pid)
		}
	}
	if f.dir == directionUnknown {
		f.dir = ref.dir
	}
	if ref.complete {
		f.complete = true
	}
	f.local.updateWith(ref.local)
	f.remote.updateWith(ref.remote)
}

func (s *state) onFlowTerminated(f *flow) {
	s.lru.remove(f)
	// Unbind this flow from its parent
	if parent, found := s.socks[f.sock]; found {
		delete(parent.flows, f.remote.addr.String())
	}
	if f.isValid() {
		s.done.add(f)
	}
	s.numFlows--
}

func (l *linkedList) add(f linkedElement) {
	if f == nil || f.Next() != nil || f.Prev() != nil {
		panic("bad flow in linked list")
	}
	l.size++
	if l.tail == nil {
		l.head = f
		l.tail = f
		f.SetNext(nil)
		f.SetPrev(nil)
		return
	}
	l.tail.SetNext(f)
	f.SetPrev(l.tail)
	l.tail = f
	f.SetNext(nil)
}

func (l *linkedList) peek() linkedElement {
	return l.head
}

func (l *linkedList) get() linkedElement {
	f := l.head
	if f != nil {
		l.remove(f)
	}
	return f
}

func (l *linkedList) remove(e linkedElement) {
	l.size--
	if e.Prev() != nil {
		e.Prev().SetNext(e.Next())
	} else {
		l.head = e.Next()
	}
	if e.Next() != nil {
		e.Next().SetPrev(e.Prev())
	} else {
		l.tail = e.Prev()
	}
	e.SetPrev(nil)
	e.SetNext(nil)
}

func (f *flow) toEvent(final bool, resolver dnsResolver) (ev mb.Event, err error) {
	localAddr := f.local.addr
	remoteAddr := f.remote.addr

	local := common.MapStr{
		"ip":      localAddr.IP.String(),
		"port":    localAddr.Port,
		"packets": f.local.packets,
		"bytes":   f.local.bytes,
	}

	remote := common.MapStr{
		"ip":      remoteAddr.IP.String(),
		"port":    remoteAddr.Port,
		"packets": f.remote.packets,
		"bytes":   f.remote.bytes,
	}

	src, dst := local, remote
	if f.dir == directionInbound {
		src, dst = dst, src
	}

	inetType := f.inetType
	// Under Linux, a socket created as AF_INET6 can receive IPv4 connections
	// and it will use the IPv4 stack.
	// This results in src and dst address using IPv4 mapped addresses (which
	// Golang converts to IPv4 automatically). It will be misleading to report
	// network.type: ipv6 and have v4 addresses, so it's better to report
	// a network.type of ipv4 (which also matches the actual stack used).
	if inetType == inetTypeIPv6 && f.local.addr.IP.To4() != nil && f.remote.addr.IP.To4() != nil {
		inetType = inetTypeIPv4
	}
	root := common.MapStr{
		"source":      src,
		"client":      src,
		"destination": dst,
		"server":      dst,
		"network": common.MapStr{
			"direction": f.dir.String(),
			"type":      inetType.String(),
			"transport": f.proto.String(),
			"packets":   f.local.packets + f.remote.packets,
			"bytes":     f.local.bytes + f.remote.bytes,
			"community_id": flowhash.CommunityID.Hash(flowhash.Flow{
				SourceIP:        localAddr.IP,
				SourcePort:      uint16(localAddr.Port),
				DestinationIP:   remoteAddr.IP,
				DestinationPort: uint16(remoteAddr.Port),
				Protocol:        uint8(f.proto),
			}),
		},
		"event": common.MapStr{
			"kind":     "event",
			"action":   "network_flow",
			"category": "network_traffic",
			"start":    f.createdTime,
			"end":      f.lastSeenTime,
			"duration": f.lastSeenTime.Sub(f.createdTime).Nanoseconds(),
		},
		"flow": common.MapStr{
			"final":    final,
			"complete": f.complete,
		},
	}

	metricset := common.MapStr{
		"kernel_sock_address": fmt.Sprintf("0x%x", f.sock),
		"internal_version":    "1.0.3",
	}

	if f.pid != 0 {
		process := common.MapStr{
			"pid": int(f.pid),
		}
		if f.process != nil {
			process["name"] = f.process.name
			process["args"] = f.process.args
			process["executable"] = f.process.path
			if f.process.createdTime != (time.Time{}) {
				process["created"] = f.process.createdTime
			}

			if f.process.hasCreds {
				uid := strconv.Itoa(int(f.process.uid))
				gid := strconv.Itoa(int(f.process.gid))
				root.Put("user.id", uid)
				root.Put("group.id", gid)
				if name := userCache.LookupUID(uid); name != "" {
					root.Put("user.name", name)
				}
				if name := groupCache.LookupGID(gid); name != "" {
					root.Put("group.name", name)
				}
				metricset["uid"] = f.process.uid
				metricset["gid"] = f.process.gid
				metricset["euid"] = f.process.euid
				metricset["egid"] = f.process.egid
			}

			if resolver != nil {
				if domain, found := resolver.ResolveIP(f.pid, f.local.addr.IP); found {
					local["domain"] = domain
				}
				if domain, found := resolver.ResolveIP(f.pid, f.remote.addr.IP); found {
					remote["domain"] = domain
				}
			}
		}
		root["process"] = process
	}

	return mb.Event{
		RootFields:      root,
		MetricSetFields: metricset,
	}, nil
}

func (s *state) SyncClocks(kernelNanos, userNanos uint64) error {
	userTime := time.Unix(int64(time.Duration(userNanos)/time.Second), int64(time.Duration(userNanos)%time.Second))
	bootTime := userTime.Add(-time.Duration(kernelNanos))
	s.Lock()
	if s.kernelEpoch == (time.Time{}) {
		s.kernelEpoch = bootTime
		s.Unlock()
		return nil
	}
	drift := s.kernelEpoch.Sub(bootTime)
	adjusted := drift < -s.clockMaxDrift || drift > s.clockMaxDrift
	if adjusted {
		s.kernelEpoch = bootTime
	}
	s.Unlock()
	if adjusted {
		s.log.Debugf("adjusted internal clock drift=%s", drift)
	}
	return nil
}

func (s *state) kernTimestampToTime(ts kernelTime) time.Time {
	if ts == 0 {
		return time.Time{}
	}
	if s.kernelEpoch == (time.Time{}) {
		// This is the first event and time sync hasn't happened yet.
		// Take a temporary epoch relative to time.Now()
		now := time.Now()
		s.kernelEpoch = now.Add(-time.Duration(ts))
		return now
	}
	return s.kernelEpoch.Add(time.Duration(ts))
}
