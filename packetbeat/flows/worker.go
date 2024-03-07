// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package flows

import (
	"encoding/binary"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/flowhash"
	"github.com/elastic/beats/v7/packetbeat/procs"
	"github.com/elastic/beats/v7/packetbeat/protos/applayer"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

var (
	ErrInvalidTimeout = errors.New("timeout must be >= 1s")
	ErrInvalidPeriod  = errors.New("report period must be -1 or >= 1s")
)

// worker is a generic asynchronous function processor.
type worker struct {
	wg   sync.WaitGroup
	done chan struct{}
	run  func(*worker)
}

// newWorker returns a handle to a worker to run fn.
func newWorker(fn func(w *worker)) *worker {
	return &worker{
		done: make(chan struct{}),
		run:  fn,
	}
}

// start starts execution of the worker function.
func (w *worker) start() {
	debugf("start flows worker")
	w.wg.Add(1)
	go func() {
		defer w.finished()
		w.run(w)
	}()
}

// finished decrements the workers working function count. finished
// must be called the same number of times as start over the lifetime
// of the worker.
func (w *worker) finished() {
	w.wg.Done()
	logp.Info("flows worker loop stopped")
}

// stop terminates the function and waits until processing is complete.
// stop may only be called once.
func (w *worker) stop() {
	debugf("stop flows worker")
	close(w.done)
	w.wg.Wait()
	debugf("stopped flows worker")
}

// sleep will sleep for the provided duration unless the worker has been
// stopped. sleep returns whether the worker can continue processing.
func (w *worker) sleep(d time.Duration) bool {
	select {
	case <-w.done:
		return false
	case <-time.After(d):
		return true
	}
}

// tick will sleep until the provided ticker fires unless the worker has been
// stopped. tick returns whether the worker can continue processing.
func (w *worker) tick(t *time.Ticker) bool {
	select {
	case <-w.done:
		return false
	case <-t.C:
		return true
	}
}

// periodically will execute fn each tick duration until the worker has been
// stopped or fn returns a non-nil error.
func (w *worker) periodically(tick time.Duration, fn func() error) {
	defer debugf("stop periodic loop")

	ticker := time.NewTicker(tick)
	defer ticker.Stop()
	for {
		cont := w.tick(ticker)
		if !cont {
			return
		}

		err := fn()
		if err != nil {
			return
		}
	}
}

// newFlowsWorker returns a worker with a flow lifetime specified by timeout and a
// reporting intervals specified by period. If period is less than or equal to zero
// reporting will be done at flow lifetime end.
// Flows are published via the pub Reporter after being enriched with process information
// by watcher.
func newFlowsWorker(pub Reporter, watcher *procs.ProcessesWatcher, table *flowMetaTable, counters *counterReg, timeout, period time.Duration, enableDeltaFlowReports bool) (*worker, error) {
	if timeout < time.Second {
		return nil, ErrInvalidTimeout
	}

	if 0 < period && period < time.Second {
		return nil, ErrInvalidPeriod
	}

	tick := timeout
	ticksTimeout := 1
	ticksPeriod := -1
	if period > 0 {
		tick = gcd(timeout, period)
		if tick < time.Second {
			tick = time.Second
		}

		ticksTimeout = int(timeout / tick)
		if ticksTimeout == 0 {
			ticksTimeout = 1
		}

		ticksPeriod = int(period / tick)
		if ticksPeriod == 0 {
			ticksPeriod = 1
		}
	}

	debugf("new flows worker. timeout=%v, period=%v, tick=%v, ticksTO=%v, ticksP=%v",
		timeout, period, tick, ticksTimeout, ticksPeriod)

	defaultBatchSize := 1024
	processor := &flowsProcessor{
		table:                    table,
		watcher:                  watcher,
		counters:                 counters,
		timeout:                  timeout,
		enableDeltaFlowReporting: enableDeltaFlowReports,
	}
	processor.spool.init(pub, defaultBatchSize)

	return makeWorker(processor, tick, ticksTimeout, ticksPeriod, 10)
}

// gcd returns the greatest common divisor of a and b.
func gcd(a, b time.Duration) time.Duration {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

// makeWorker returns a worker that runs processor.execute each tick. Each timeout'th tick,
// the worker will check flow timeouts and each period'th tick, the worker will report flow
// events to be published.
func makeWorker(processor *flowsProcessor, tick time.Duration, timeout, period int, align int64) (*worker, error) {
	return newWorker(func(w *worker) {
		defer processor.execute(w, false, true, true)

		if align > 0 {
			// Wait until the current time rounded up to nearest align seconds.
			aligned := time.Unix(((time.Now().Unix()+(align-1))/align)*align, 0)
			waitStart := time.Until(aligned)
			debugf("worker wait start(%v): %v", aligned, waitStart)
			if cont := w.sleep(waitStart); !cont {
				return
			}
		}

		nTimeout := timeout
		nPeriod := period
		reportPeriodically := period > 0
		debugf("start flows worker loop")
		w.periodically(tick, func() error {
			nTimeout--
			nPeriod--
			debugf("worker tick, nTimeout=%v, nPeriod=%v", nTimeout, nPeriod)

			handleTimeout := nTimeout == 0
			if handleTimeout {
				nTimeout = timeout
			}
			handleReports := reportPeriodically && nPeriod == 0
			if nPeriod <= 0 {
				nPeriod = period
			}

			processor.execute(w, handleTimeout, handleReports, false)
			return nil
		})
	}), nil
}

type flowsProcessor struct {
	spool                    spool
	watcher                  *procs.ProcessesWatcher
	table                    *flowMetaTable
	counters                 *counterReg
	timeout                  time.Duration
	enableDeltaFlowReporting bool
}

func (fw *flowsProcessor) execute(w *worker, checkTimeout, handleReports, lastReport bool) {
	if !checkTimeout && !handleReports {
		return
	}

	debugf("exec tick, timeout=%v, report=%v", checkTimeout, handleReports)

	// get counter names snapshot if reports must be generated
	fw.counters.mutex.Lock()
	intNames := fw.counters.ints.getNames()
	uintNames := fw.counters.uints.getNames()
	floatNames := fw.counters.floats.getNames()
	fw.counters.mutex.Unlock()

	fw.table.Lock()
	defer fw.table.Unlock()
	ts := time.Now()

	// TODO: create snapshot inside flows/tables, so deletion of timed-out flows
	//       and reporting flows stats can be done more concurrent to packet
	//       processing.

	for table := fw.table.tables.head; table != nil; table = table.next {
		var next *biFlow
		for flow := table.flows.head; flow != nil; flow = next {
			next = flow.next

			debugf("handle flow: %v, %v", flow.id.flowIDMeta, flow.id.flowID)

			reportFlow := handleReports
			isOver := lastReport
			if checkTimeout {
				if ts.Sub(flow.ts) > fw.timeout {
					debugf("kill flow")

					reportFlow = true
					flow.kill() // mark flow as killed
					isOver = true
					table.remove(flow)
				}
			}

			if reportFlow {
				debugf("report flow")
				fw.report(w, ts, flow, isOver, intNames, uintNames, floatNames)
			}
		}
	}

	fw.spool.flush()
}

func (fw *flowsProcessor) report(w *worker, ts time.Time, flow *biFlow, isOver bool, intNames, uintNames, floatNames []string) {
	event := createEvent(fw.watcher, ts, flow, isOver, intNames, uintNames, floatNames, fw.enableDeltaFlowReporting)

	debugf("add event: %v", event)
	fw.spool.publish(event)
}

func createEvent(watcher *procs.ProcessesWatcher, ts time.Time, f *biFlow, isOver bool, intNames, uintNames, floatNames []string, enableDeltaFlowReporting bool) beat.Event {
	timestamp := ts

	event := mapstr.M{
		"start":    common.Time(f.createTS),
		"end":      common.Time(f.ts),
		"duration": f.ts.Sub(f.createTS),
		"dataset":  "flow",
		"kind":     "event",
		"category": []string{"network"},
		"action":   "network_flow",
	}
	eventType := []string{"connection"}
	if isOver {
		eventType = append(eventType, "end")
	}
	event["type"] = eventType

	flow := mapstr.M{
		"id":    common.NetString(f.id.Serialize()),
		"final": isOver,
	}
	fields := mapstr.M{
		"event": event,
		"flow":  flow,
		"type":  "flow",
	}
	network := mapstr.M{}
	source := mapstr.M{}
	dest := mapstr.M{}
	tuple := common.IPPortTuple{}
	var communityID flowhash.Flow
	var proto applayer.Transport

	// add ethernet layer meta data
	if src, dst, ok := f.id.EthAddr(); ok {
		source["mac"] = formatHardwareAddr(net.HardwareAddr(src))
		dest["mac"] = formatHardwareAddr(net.HardwareAddr(dst))
	}

	// add vlan
	if vlan := f.id.OutterVLan(); vlan != nil {
		vlanID := uint64(binary.LittleEndian.Uint16(vlan))
		putOrAppendUint64(flow, "vlan", vlanID)
	}
	if vlan := f.id.VLan(); vlan != nil {
		vlanID := uint64(binary.LittleEndian.Uint16(vlan))
		putOrAppendUint64(flow, "vlan", vlanID)
	}

	// ipv4 layer meta data
	if src, dst, ok := f.id.OutterIPv4Addr(); ok {
		srcIP, dstIP := net.IP(src), net.IP(dst)
		source["ip"] = srcIP.String()
		dest["ip"] = dstIP.String()
		tuple.SrcIP = srcIP
		tuple.DstIP = dstIP
		tuple.IPLength = 4
		network["type"] = "ipv4"
		communityID.SourceIP = srcIP
		communityID.DestinationIP = dstIP
	}
	if src, dst, ok := f.id.IPv4Addr(); ok {
		srcIP, dstIP := net.IP(src), net.IP(dst)
		putOrAppendString(source, "ip", srcIP.String())
		putOrAppendString(dest, "ip", dstIP.String())
		// Save IPs for process matching if an outer layer was not present
		if tuple.IPLength == 0 {
			tuple.SrcIP = srcIP
			tuple.DstIP = dstIP
			tuple.IPLength = 4
			communityID.SourceIP = srcIP
			communityID.DestinationIP = dstIP
			network["type"] = "ipv4"
		}
	}

	// ipv6 layer meta data
	if src, dst, ok := f.id.OutterIPv6Addr(); ok {
		srcIP, dstIP := net.IP(src), net.IP(dst)
		putOrAppendString(source, "ip", srcIP.String())
		putOrAppendString(dest, "ip", dstIP.String())
		tuple.SrcIP = srcIP
		tuple.DstIP = dstIP
		tuple.IPLength = 6
		network["type"] = "ipv6"
		communityID.SourceIP = srcIP
		communityID.DestinationIP = dstIP
	}
	if src, dst, ok := f.id.IPv6Addr(); ok {
		srcIP, dstIP := net.IP(src), net.IP(dst)
		putOrAppendString(source, "ip", srcIP.String())
		putOrAppendString(dest, "ip", dstIP.String())
		// Save IPs for process matching if an outer layer was not present
		if tuple.IPLength == 0 {
			tuple.SrcIP = srcIP
			tuple.DstIP = dstIP
			tuple.IPLength = 6
			communityID.SourceIP = srcIP
			communityID.DestinationIP = dstIP
			network["type"] = "ipv6"
		}
	}

	// udp layer meta data
	if src, dst, ok := f.id.UDPAddr(); ok {
		tuple.SrcPort = binary.LittleEndian.Uint16(src)
		tuple.DstPort = binary.LittleEndian.Uint16(dst)
		source["port"], dest["port"] = tuple.SrcPort, tuple.DstPort
		network["transport"] = "udp"
		proto = applayer.TransportUDP
		communityID.SourcePort = tuple.SrcPort
		communityID.DestinationPort = tuple.DstPort
		communityID.Protocol = 17
	}

	// tcp layer meta data
	if src, dst, ok := f.id.TCPAddr(); ok {
		tuple.SrcPort = binary.LittleEndian.Uint16(src)
		tuple.DstPort = binary.LittleEndian.Uint16(dst)
		source["port"], dest["port"] = tuple.SrcPort, tuple.DstPort
		network["transport"] = "tcp"
		proto = applayer.TransportTCP
		communityID.SourcePort = tuple.SrcPort
		communityID.DestinationPort = tuple.DstPort
		communityID.Protocol = 6
	}

	var totalBytes, totalPackets uint64
	if f.stats[0] != nil {
		// Source stats.
		stats := encodeStats(f.stats[0], intNames, uintNames, floatNames, enableDeltaFlowReporting)
		for k, v := range stats {
			switch k {
			case "icmpV4TypeCode":
				if typeCode, ok := v.(uint64); ok && typeCode > 0 {
					network["transport"] = "icmp"
					communityID.Protocol = 1
					communityID.ICMP.Type = uint8(typeCode >> 8)
					communityID.ICMP.Code = uint8(typeCode)
				}
			case "icmpV6TypeCode":
				if typeCode, ok := v.(uint64); ok && typeCode > 0 {
					network["transport"] = "ipv6-icmp"
					communityID.Protocol = 58
					communityID.ICMP.Type = uint8(typeCode >> 8)
					communityID.ICMP.Code = uint8(typeCode)
				}
			default:
				source[k] = v
			}
		}

		if v, found := stats["bytes"]; found {
			totalBytes += v.(uint64)
		}
		if v, found := stats["packets"]; found {
			totalPackets += v.(uint64)
		}
	}
	if f.stats[1] != nil {
		// Destination stats.
		stats := encodeStats(f.stats[1], intNames, uintNames, floatNames, enableDeltaFlowReporting)
		for k, v := range stats {
			switch k {
			case "icmpV4TypeCode", "icmpV6TypeCode":
			default:
				dest[k] = v
			}
		}

		if v, found := stats["bytes"]; found {
			totalBytes += v.(uint64)
		}
		if v, found := stats["packets"]; found {
			totalPackets += v.(uint64)
		}
	}
	if communityID.Protocol > 0 && len(communityID.SourceIP) > 0 && len(communityID.DestinationIP) > 0 {
		hash := flowhash.CommunityID.Hash(communityID)
		network["community_id"] = hash
	}
	network["bytes"] = totalBytes
	network["packets"] = totalPackets
	fields["network"] = network

	// Set process information if it's available
	if tuple.IPLength != 0 && tuple.SrcPort != 0 {
		if proc := watcher.FindProcessesTuple(&tuple, proto); proc != nil {
			if proc.Src.PID > 0 {
				p := mapstr.M{
					"pid":               proc.Src.PID,
					"name":              proc.Src.Name,
					"args":              proc.Src.Args,
					"ppid":              proc.Src.PPID,
					"executable":        proc.Src.Exe,
					"start":             proc.Src.StartTime,
					"working_directory": proc.Src.CWD,
				}
				if proc.Src.CWD != "" {
					p["working_directory"] = proc.Src.CWD
				}
				source["process"] = p
				fields["process"] = p
			}
			if proc.Dst.PID > 0 {
				p := mapstr.M{
					"pid":               proc.Dst.PID,
					"name":              proc.Dst.Name,
					"args":              proc.Dst.Args,
					"ppid":              proc.Dst.PPID,
					"executable":        proc.Dst.Exe,
					"start":             proc.Dst.StartTime,
					"working_directory": proc.Src.CWD,
				}
				if proc.Dst.CWD != "" {
					p["working_directory"] = proc.Dst.CWD
				}
				dest["process"] = p
				fields["process"] = p
			}
		}
	}

	fields["source"] = source
	fields["destination"] = dest

	return beat.Event{
		Timestamp: timestamp,
		Fields:    fields,
	}
}

// formatHardwareAddr formats hardware addresses according to the ECS spec.
func formatHardwareAddr(addr net.HardwareAddr) string {
	buf := make([]byte, 0, len(addr)*3-1)
	for _, b := range addr {
		if len(buf) != 0 {
			buf = append(buf, '-')
		}
		const hexDigit = "0123456789ABCDEF"
		buf = append(buf, hexDigit[b>>4], hexDigit[b&0xf])
	}
	return string(buf)
}

func encodeStats(stats *flowStats, ints, uints, floats []string, enableDeltaFlowReporting bool) map[string]interface{} {
	report := make(map[string]interface{})

	i := 0
	for _, mask := range stats.intFlags {
		for m := mask; m != 0; m >>= 1 {
			if (m & 1) == 1 {
				report[ints[i]] = stats.ints[i]
			}
			i++
		}
	}

	i = 0
	for _, mask := range stats.uintFlags {
		for m := mask; m != 0; m >>= 1 {
			if (m & 1) == 1 {
				report[uints[i]] = stats.uints[i]
				if enableDeltaFlowReporting && (uints[i] == "bytes" || uints[i] == "packets") {
					// If Delta Flow Reporting is enabled, reset bytes and packets at each period.
					// Only the bytes and packets recieved during the flow period will be reported.
					// This should be thread safe as it is called under the flowmetadatatable lock.
					stats.uints[i] = 0
				}
			}
			i++
		}
	}

	i = 0
	for _, mask := range stats.floatFlags {
		for m := mask; m != 0; m >>= 1 {
			if (m & 1) == 1 {
				report[floats[i]] = stats.floats[i]
			}
			i++
		}
	}

	return report
}

func putOrAppendString(m mapstr.M, key, value string) {
	old, found := m[key]
	if !found {
		m[key] = value
		return
	}

	if old != nil {
		switch v := old.(type) {
		case string:
			m[key] = []string{v, value}
		case []string:
			m[key] = append(v, value)
		}
	}
}

func putOrAppendUint64(m mapstr.M, key string, value uint64) {
	old, found := m[key]
	if !found {
		m[key] = value
		return
	}

	if old != nil {
		switch v := old.(type) {
		case uint8:
			m[key] = []uint64{uint64(v), value}
		case uint16:
			m[key] = []uint64{uint64(v), value}
		case uint32:
			m[key] = []uint64{uint64(v), value}
		case uint64:
			m[key] = []uint64{v, value}
		case []uint64:
			m[key] = append(v, value)
		}
	}
}

// spool is an event publisher spool.
type spool struct {
	pub    Reporter
	events []beat.Event
}

// init sets the destination and spool size.
func (s *spool) init(pub Reporter, sz int) {
	s.pub = pub
	s.events = make([]beat.Event, 0, sz)
}

// publish queues the event for publication, flushing to the destination
// if the spool is full.
func (s *spool) publish(event beat.Event) {
	s.events = append(s.events, event)
	if len(s.events) == cap(s.events) {
		s.flush()
	}
}

// flush sends the spooled events to the destination and clears them
// from the spool.
func (s *spool) flush() {
	if len(s.events) == 0 {
		return
	}
	s.pub(s.events)
	// A newly allocated spool is created since the
	// elements of s.events are no longer owned by s
	// during testing and mutating them causes a panic.
	//
	// The beat.Client interface which Reporter is
	// derived from is silent on whether the caller
	// is allowed to modify elements of the slice
	// after the call to the PublishAll method returns.
	s.events = make([]beat.Event, 0, cap(s.events))
}
