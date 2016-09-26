// +build windows

package eventlog

import (
	"testing"

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
	eventlog, err := newWinEventLog(map[string]interface{}{"name": providerName, "batch_read_size": batchReadSize})
	if err != nil {
		t.Fatal(err)
	}
	err = eventlog.Open(0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := eventlog.Close()
		if err != nil {
			t.Fatal(err)
		}
	}()

	records, err := eventlog.Read()
	if err != nil {
		t.Fatal(err)
	}

	assert.Len(t, records, batchReadSize)
}
