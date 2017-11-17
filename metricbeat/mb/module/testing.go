package module

import (
	"encoding/json"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/testing"
)

// testingReporter offers reported interface and send results to testing.Driver
type testingReporter struct {
	driver testing.Driver
	done   <-chan struct{}
}

func (r *testingReporter) Done() <-chan struct{} {
	return r.done
}

func (r *testingReporter) Event(event common.MapStr) bool {
	return r.ErrorWith(nil, event)
}

func (r *testingReporter) Error(err error) bool {
	return r.ErrorWith(err, nil)
}

func (r *testingReporter) ErrorWith(err error, event common.MapStr) bool {
	if err != nil {
		r.driver.Error("error", err)
	}

	if event != nil {
		d, err := json.MarshalIndent(&event, "", " ")
		if err != nil {
			r.driver.Error("convert event", err)
			return true
		}

		r.driver.Result(string(d))
	}

	return true
}

func (r testingReporter) StartFetchTimer() {}
