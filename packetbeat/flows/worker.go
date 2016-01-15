package flows

import (
	"errors"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/publisher"
)

type flowsProcessor struct {
	pub publisher.Client

	table    *flowMetaTable
	counters *counterReg
	timeout  time.Duration

	events []common.MapStr
}

var (
	ErrInvalidTimeout = errors.New("timeout must not >= 1s")
	ErrInvalidPeriod  = errors.New("report period must be -1 or >= 1s")
)

func newFlowsWorker(
	pub publisher.Client,
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
		ticksTimeout = int(timeout / tickDuration)
		ticksPeriod = int(period / tickDuration)
	}

	processor := &flowsProcessor{
		pub:      pub,
		table:    table,
		counters: counters,
		timeout:  timeout,
	}

	return newWorker(func(w *worker) {
		defer w.finished()

		// round time to nearest 10 seconds for alignment
		aligned := time.Unix((time.Now().Unix()+9/10)*10, 0)
		if cont := w.sleep(aligned.Sub(time.Now())); !cont {
			return
		}

		nTimeout := ticksTimeout
		nPeriod := ticksPeriod
		reportPeriodically := ticksPeriod > 0
		w.periodicaly(tickDuration, func() error {
			nTimeout--
			nPeriod--

			handleTimeout := nTimeout == 0
			handleReports := reportPeriodically && nPeriod == 0
			if handleTimeout {
				nTimeout = ticksTimeout
			}
			if handleReports {
				nPeriod = ticksPeriod
			}

			processor.execute(w, handleTimeout, handleReports)
			return nil
		})
	}), nil
}

func (fw *flowsProcessor) execute(w *worker, checkTimeout, handleReports bool) {
	if !checkTimeout && !handleReports {
		return
	}

	// get counter names snapshot if reports must be generated
	var intNames []string
	var floatNames []string
	if handleReports {
		fw.counters.mutex.Lock()
		intNames = fw.counters.ints.getNames()
		floatNames = fw.counters.floats.getNames()
		fw.counters.mutex.Unlock()
	}

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

			reportFlow := handleReports
			if checkTimeout {
				if ts.Sub(flow.ts) > fw.timeout {
					reportFlow = true

					flow.kill() // mark flow as killed
					table.remove(flow)
				}
			}

			if !reportFlow {
				fw.report(w, ts, flow, intNames, floatNames)
			}
		}
	}

	fw.flush(w)
}

func (fw *flowsProcessor) report(
	w *worker,
	ts time.Time,
	flow *biFlow,
	intNames, floatNames []string,
) {
	defaultBatchSize := 1024

	event := createEvent(ts, flow, intNames, floatNames)
	if event == nil {
		return
	}

	if fw.events == nil {
		fw.events = make([]common.MapStr, 0, defaultBatchSize)
	}

	fw.events = append(fw.events, event)
	if len(fw.events) == cap(fw.events) {
		fw.flush(w)
	}
}

func (fw *flowsProcessor) flush(w *worker) {
	if fw.events == nil {
		return
	}

	fw.pub.PublishEvents(fw.events)
	fw.events = nil
}

func createEvent(ts time.Time, f *biFlow, intNames, floatNames []string) common.MapStr {
	// TODO: create event
	return nil
}
