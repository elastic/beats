package eventlog

import (
	"testing"

	"github.com/elastic/beats/libbeat/common"
)

type factory func(*common.Config) (EventLog, error)
type teardown func()

func fatalErr(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}

func newTestEventLog(t *testing.T, factory factory, options map[string]interface{}) EventLog {
	config, err := common.NewConfigFrom(options)
	fatalErr(t, err)
	eventLog, err := factory(config)
	fatalErr(t, err)
	return eventLog
}

func setupEventLog(t *testing.T, factory factory, recordID uint64, options map[string]interface{}) (EventLog, teardown) {
	eventLog := newTestEventLog(t, factory, options)
	fatalErr(t, eventLog.Open(recordID))
	return eventLog, func() { fatalErr(t, eventLog.Close()) }
}
