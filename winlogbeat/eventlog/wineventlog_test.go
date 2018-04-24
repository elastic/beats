// +build windows

package eventlog

import (
	"expvar"
	"strconv"
	"testing"

	elog "github.com/andrewkroh/sys/windows/svc/eventlog"
	"github.com/stretchr/testify/assert"
)

func TestWinEventLogBatchReadSize(t *testing.T) {
	configureLogp()
	log, err := initLog(providerName, sourceName, eventCreateMsgFile)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := uninstallLog(providerName, sourceName, log)
		if err != nil {
			t.Fatal(err)
		}
	}()

	// Publish test messages:
	for k, m := range messages {
		err = log.Report(m.eventType, k, []string{m.message})
		if err != nil {
			t.Fatal(err)
		}
	}

	batchReadSize := 2
	eventlog, teardown := setupWinEventLog(t, 0, map[string]interface{}{"name": providerName, "batch_read_size": batchReadSize})
	defer teardown()

	records, err := eventlog.Read()
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, records, batchReadSize)
}

// TestReadLargeBatchSize tests reading from an event log using a large
// read_batch_size parameter. When combined with large messages this causes
// EvtNext (wineventlog.EventRecords) to fail with RPC_S_INVALID_BOUND error.
func TestReadLargeBatchSize(t *testing.T) {
	configureLogp()
	log, err := initLog(providerName, sourceName, eventCreateMsgFile)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := uninstallLog(providerName, sourceName, log)
		if err != nil {
			t.Fatal(err)
		}
	}()

	setLogSize(t, providerName, gigabyte)

	// Publish large test messages.
	totalEvents := 1000
	for i := 0; i < totalEvents; i++ {
		err = log.Report(elog.Info, uint32(i%1000), []string{strconv.Itoa(i) + " " + randomSentence(31800)})
		if err != nil {
			t.Fatal("ReportEvent error", err)
		}
	}

	eventlog, teardown := setupWinEventLog(t, 0, map[string]interface{}{"name": providerName, "batch_read_size": 1024})
	defer teardown()

	var eventCount int
	for eventCount < totalEvents {
		records, err := eventlog.Read()
		if err != nil {
			t.Fatal("read error", err)
		}
		if len(records) == 0 {
			t.Fatal("read returned 0 records")
		}
		eventCount += len(records)
	}

	t.Logf("number of records returned: %v", eventCount)

	wineventlog := eventlog.(*winEventLog)
	assert.Equal(t, 1024, wineventlog.maxRead)

	expvar.Do(func(kv expvar.KeyValue) {
		if kv.Key == "read_errors" {
			t.Log(kv)
		}
	})
}

func setupWinEventLog(t *testing.T, recordID uint64, options map[string]interface{}) (EventLog, func()) {
	return setupEventLog(t, newWinEventLog, recordID, options)
}
