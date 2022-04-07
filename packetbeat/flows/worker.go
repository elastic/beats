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
	"time"

	"github.com/elastic/beats/v8/libbeat/beat"
	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/common/flowhash"
	"github.com/elastic/beats/v8/packetbeat/procs"
	"github.com/elastic/beats/v8/packetbeat/protos/applayer"
)

type flowsProcessor struct {
	spool    spool
	watcher  procs.ProcessesWatcher
	table    *flowMetaTable
	counters *counterReg
	timeout  time.Duration
}

var (
	ErrInvalidTimeout = errors.New("timeout must be >= 1s")
	ErrInvalidPeriod  = errors.New("report period must be -1 or >= 1s")
)

func newFlowsWorker(
	pub Reporter,
	watcher procs.ProcessesWatcher,
	table *flowMetaTable,
	counters *counterReg,
	timeout, period time.Duration,
) (*worker, error) {
	oneSecond := 1 * time.Second

	if timeout < oneSecond {
		return nil, ErrInvalidTimeout
	}

	if 0 < period && period < oneSecond {
		return nil, ErrInvalidPeriod
	}

	tickDuration := timeout
	ticksTimeout := 1
	ticksPeriod := -1
	if period > 0 {
		tickDuration = time.Duration(gcd(int64(timeout), int64(period)))
		if tickDuration < oneSecond {
			tickDuration = oneSecond
		}

		ticksTimeout = int(timeout / tickDuration)
		if ticksTimeout == 0 {
			ticksTimeout = 1
		}

		ticksPeriod = int(period / tickDuration)
		if ticksPeriod == 0 {
			ticksPeriod = 1
		}
	}

	debugf("new flows worker. timeout=%v, period=%v, tick=%v, ticksTO=%v, ticksP=%v",
		timeout, period, tickDuration, ticksTimeout, ticksPeriod)

	defaultBatchSize := 1024
	processor := &flowsProcessor{
		table:    table,
		watcher:  watcher,
		counters: counters,
		timeout:  timeout,
	}
	processor.spool.init(pub, defaultBatchSize)

	return makeWorker(processor, tickDuration, ticksTimeout, ticksPeriod, 10)
}

func makeWorker(
	processor *flowsProcessor,
	tickDuration time.Duration,
	ticksTimeout, ticksPeriod int,
	align int64,
) (*worker, error) {
	return newWorker(func(w *worker) {
		defer processor.execute(w, false, true, true)

		if align > 0 {
			// round time to nearest 10 seconds for alignment
			aligned := time.Unix(((time.Now().Unix()+(align-1))/align)*align, 0)
			waitStart := time.Until(aligned)
			debugf("worker wait start(%v): %v", aligned, waitStart)
			if cont := w.sleep(waitStart); !cont {
				return
			}
		}

		nTimeout := ticksTimeout
		nPeriod := ticksPeriod
		reportPeriodically := ticksPeriod > 0
		debugf("start flows worker loop")
		w.periodically(tickDuration, func() error {
			nTimeout--
			nPeriod--
			debugf("worker tick, nTimeout=%v, nPeriod=%v", nTimeout, nPeriod)

			handleTimeout := nTimeout == 0
			handleReports := reportPeriodically && nPeriod == 0
			if handleTimeout {
				nTimeout = ticksTimeout
			}
			if nPeriod <= 0 {
				nPeriod = ticksPeriod
			}

			processor.execute(w, handleTimeout, handleReports, false)
			return nil
		})
	}), nil
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

func (fw *flowsProcessor) report(
	w *worker,
	ts time.Time,
	flow *biFlow,
	isOver bool,
	intNames, uintNames, floatNames []string,
) {
	event := createEvent(fw.watcher, ts, flow, isOver, intNames, uintNames, floatNames)

	debugf("add event: %v", event)
	fw.spool.publish(event)
}

func createEvent(
	watcher procs.ProcessesWatcher,
	ts time.Time, f *biFlow,
	isOver bool,
	intNames, uintNames, floatNames []string,
) beat.Event {
	timestamp := ts

	event := common.MapStr{
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

	flow := common.MapStr{
		"id":    common.NetString(f.id.Serialize()),
		"final": isOver,
	}
	fields := common.MapStr{
		"event": event,
		"flow":  flow,
		"type":  "flow",
	}
	network := common.MapStr{}
	source := common.MapStr{}
	dest := common.MapStr{}
	tuple := common.IPPortTuple{}
	var communityID flowhash.Flow
	var proto applayer.Transport

	// add ethernet layer meta data
	if src, dst, ok := f.id.EthAddr(); ok {
		source["mac"] = net.HardwareAddr(src).String()
		dest["mac"] = net.HardwareAddr(dst).String()
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
		stats := encodeStats(f.stats[0], intNames, uintNames, floatNames)
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
		stats := encodeStats(f.stats[1], intNames, uintNames, floatNames)
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
				p := common.MapStr{
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
				p := common.MapStr{
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

func encodeStats(
	stats *flowStats,
	ints, uints, floats []string,
) map[string]interface{} {
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

func putOrAppendString(m common.MapStr, key, value string) {
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

func putOrAppendUint64(m common.MapStr, key string, value uint64) {
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
			m[key] = []uint64{uint64(v), value}
		case []uint64:
			m[key] = append(v, value)
		}
	}
}
