package flows

import (
	"encoding/base64"
	"encoding/binary"
	"errors"
	"net"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

type flowsProcessor struct {
	spool    spool
	table    *flowMetaTable
	counters *counterReg
	timeout  time.Duration
}

var (
	ErrInvalidTimeout = errors.New("timeout must not <= 1s")
	ErrInvalidPeriod  = errors.New("report period must be -1 or >= 1s")
)

func newFlowsWorker(
	pub Reporter,
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
			waitStart := aligned.Sub(time.Now())
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

	// TODO: create snapshot inside flows/tables, so deletion of timedout flows
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
	event := createEvent(ts, flow, isOver, intNames, uintNames, floatNames)

	debugf("add event: %v", event)
	fw.spool.publish(event)
}

func createEvent(
	ts time.Time, f *biFlow,
	isOver bool,
	intNames, uintNames, floatNames []string,
) beat.Event {
	timestamp := ts
	fields := common.MapStr{
		"start_time": common.Time(f.createTS),
		"last_time":  common.Time(f.ts),
		"type":       "flow",
		"flow_id":    common.NetString(f.id.Serialize()),
		"final":      isOver,
	}

	source := common.MapStr{}
	dest := common.MapStr{}

	// add ethernet layer meta data
	if src, dst, ok := f.id.EthAddr(); ok {
		source["mac"] = net.HardwareAddr(src).String()
		dest["mac"] = net.HardwareAddr(dst).String()
	}

	// add vlan
	if vlan := f.id.OutterVLan(); vlan != nil {
		fields["outer_vlan"] = binary.LittleEndian.Uint16(vlan)
	}
	if vlan := f.id.VLan(); vlan != nil {
		fields["vlan"] = binary.LittleEndian.Uint16(vlan)
	}

	// add icmp
	if icmp := f.id.ICMPv4(); icmp != nil {
		fields["icmp_id"] = binary.LittleEndian.Uint16(icmp)
	} else if icmp := f.id.ICMPv6(); icmp != nil {
		fields["icmp_id"] = binary.LittleEndian.Uint16(icmp)
	}

	// ipv4 layer meta data
	if src, dst, ok := f.id.OutterIPv4Addr(); ok {
		source["outer_ip"] = net.IP(src).String()
		dest["outer_ip"] = net.IP(dst).String()
	}
	if src, dst, ok := f.id.IPv4Addr(); ok {
		source["ip"] = net.IP(src).String()
		dest["ip"] = net.IP(dst).String()
	}

	// ipv6 layer meta data
	if src, dst, ok := f.id.OutterIPv6Addr(); ok {
		source["outer_ipv6"] = net.IP(src).String()
		dest["outer_ipv6"] = net.IP(dst).String()
	}
	if src, dst, ok := f.id.IPv6Addr(); ok {
		source["ipv6"] = net.IP(src).String()
		dest["ipv6"] = net.IP(dst).String()
	}

	// udp layer meta data
	if src, dst, ok := f.id.UDPAddr(); ok {
		source["port"] = binary.LittleEndian.Uint16(src)
		dest["port"] = binary.LittleEndian.Uint16(dst)
		fields["transport"] = "udp"
	}

	// tcp layer meta data
	if src, dst, ok := f.id.TCPAddr(); ok {
		source["port"] = binary.LittleEndian.Uint16(src)
		dest["port"] = binary.LittleEndian.Uint16(dst)
		fields["transport"] = "tcp"
	}

	if id := f.id.ConnectionID(); id != nil {
		fields["connection_id"] = base64.StdEncoding.EncodeToString(id)
	}

	if f.stats[0] != nil {
		source["stats"] = encodeStats(f.stats[0], intNames, uintNames, floatNames)
	}
	if f.stats[1] != nil {
		dest["stats"] = encodeStats(f.stats[1], intNames, uintNames, floatNames)
	}

	fields["source"] = source
	fields["dest"] = dest

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
