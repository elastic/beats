// +build !integration

package lb

import (
	"errors"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/mode"
	"github.com/elastic/beats/libbeat/outputs/mode/modetest"
)

func TestAsyncLBStartStop(t *testing.T) {
	mode, _ := NewAsync(
		[]mode.AsyncProtocolClient{},
		1,
		100*time.Millisecond,
		100*time.Millisecond,
		1*time.Second,
	)
	modetest.TestMode(t, mode, testNoOpts, nil, nil, nil)
}

func testAsyncLBFailSendWithoutActiveConnection(t *testing.T, events []modetest.EventInfo) {
	enableLogging([]string{"*"})

	errFail := errors.New("fail connect")
	mode, _ := NewAsync(
		modetest.AsyncClients(2, &modetest.MockClient{
			CBConnect: modetest.ConnectFail(errFail),
		}),
		2,
		100*time.Millisecond,
		100*time.Millisecond,
		1*time.Second,
	)
	modetest.TestMode(t, mode, testNoOpts, events, modetest.Signals(false), nil)
}

func TestAsyncLBFailSendWithoutActiveConnections(t *testing.T) {
	testAsyncLBFailSendWithoutActiveConnection(t, modetest.SingleEvent(testEvent))
}

func TestAsyncLBFailSendMultWithoutActiveConnections(t *testing.T) {
	testAsyncLBFailSendWithoutActiveConnection(t, modetest.MultiEvent(2, testEvent))
}

func testAsyncLBOKSend(t *testing.T, events []modetest.EventInfo) {
	enableLogging([]string{"*"})

	var collected [][]outputs.Data
	mode, _ := NewAsync(
		modetest.AsyncClients(1, &modetest.MockClient{
			CBAsyncPublish: modetest.AsyncPublishCollect(&collected),
		}),
		2,
		100*time.Millisecond,
		100*time.Millisecond,
		1*time.Second,
	)
	modetest.TestMode(t, mode, testNoOpts, events, modetest.Signals(true), &collected)
}

func TestAsyncLBOKSend(t *testing.T) {
	testAsyncLBOKSend(t, modetest.SingleEvent(testEvent))
}

func TestAsyncLBOKSendMult(t *testing.T) {
	testAsyncLBOKSend(t, modetest.MultiEvent(10, testEvent))
}

func testAsyncLBFlakyConnectionOkSend(t *testing.T, events []modetest.EventInfo) {
	var collected [][]outputs.Data
	tmpl := &modetest.MockClient{
		Connected:      true,
		CBAsyncPublish: modetest.AsyncPublishCollectAfterFailStart(1, &collected),
	}
	mode, _ := NewAsync(
		modetest.AsyncClients(2, tmpl),
		3,
		100*time.Millisecond,
		100*time.Millisecond,
		1*time.Second,
	)
	modetest.TestMode(t, mode, testNoOpts, events, modetest.Signals(true), &collected)
}

func TestAsyncLBFlakyConnectionOkSend(t *testing.T) {
	testAsyncLBFlakyConnectionOkSend(t, modetest.SingleEvent(testEvent))
}

func TestAsyncLBFlakyConnectionOkSendMult(t *testing.T) {
	testAsyncLBFlakyConnectionOkSend(t, modetest.MultiEvent(10, testEvent))
}

func testAsyncLBFlakyFail(t *testing.T, events []modetest.EventInfo) {
	enableLogging([]string{"*"})

	var collected [][]outputs.Data
	err := errors.New("flaky")
	mode, _ := NewAsync(
		modetest.AsyncClients(2, &modetest.MockClient{
			Connected:      true,
			CBAsyncPublish: modetest.AsyncPublishCollectAfterFailStartWith(3, err, &collected),
		}),
		3,
		100*time.Millisecond,
		100*time.Millisecond,
		1*time.Second,
	)
	modetest.TestMode(t, mode, testNoOpts, events, modetest.Signals(false), &collected)
}

func TestAsyncLBFlakyFail(t *testing.T) {
	testAsyncLBFlakyFail(t, modetest.SingleEvent(testEvent))
}

func TestAsyncLBMultiFlakyFail(t *testing.T) {
	testAsyncLBFlakyFail(t, modetest.MultiEvent(10, testEvent))
}

func testAsyncLBTemporayFailure(t *testing.T, events []modetest.EventInfo) {
	var collected [][]outputs.Data
	mode, _ := NewAsync(
		modetest.AsyncClients(1, &modetest.MockClient{
			Connected: true,
			CBAsyncPublish: modetest.AsyncPublishCollectAfterFailStartWith(
				1, mode.ErrTempBulkFailure, &collected),
		}),
		3,
		100*time.Millisecond,
		100*time.Millisecond,
		1*time.Second,
	)
	modetest.TestMode(t, mode, testNoOpts, events, modetest.Signals(true), &collected)
}

func TestAsyncLBTemporayFailure(t *testing.T) {
	testAsyncLBTemporayFailure(t, modetest.SingleEvent(testEvent))
}

func TestAsyncLBTemporayFailureMutlEvents(t *testing.T) {
	testAsyncLBTemporayFailure(t, modetest.MultiEvent(10, testEvent))
}

func testAsyncLBTempFlakyFail(t *testing.T, events []modetest.EventInfo) {
	enableLogging([]string{"*"})

	var collected [][]outputs.Data
	mode, _ := NewAsync(
		modetest.AsyncClients(2, &modetest.MockClient{
			Connected: true,
			CBAsyncPublish: modetest.AsyncPublishCollectAfterFailStartWith(
				6, mode.ErrTempBulkFailure, &collected),
		}),
		3,
		100*time.Millisecond,
		100*time.Millisecond,
		1*time.Second,
	)
	modetest.TestMode(t, mode, testNoOpts, events, modetest.Signals(false), &collected)
}

func TestAsyncLBTempFlakyFail(t *testing.T) {
	testAsyncLBTempFlakyFail(t, modetest.SingleEvent(testEvent))
}

func TestAsyncLBMultiTempFlakyFail(t *testing.T) {
	testAsyncLBTempFlakyFail(t, modetest.MultiEvent(10, testEvent))
}

func testAsyncLBFlakyInfAttempts(t *testing.T, events []modetest.EventInfo) {
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"*"})
	}

	var collected [][]outputs.Data
	err := errors.New("flaky")
	mode, _ := NewAsync(
		modetest.AsyncClients(2, &modetest.MockClient{
			Connected: true,
			CBAsyncPublish: modetest.AsyncPublishCollectAfterFailStartWith(
				50, err, &collected),
		}),
		0,
		1*time.Nanosecond,
		1*time.Millisecond,
		4*time.Millisecond,
	)
	modetest.TestMode(t, mode, testNoOpts, events, modetest.Signals(true), &collected)
}

func TestAsyncLBFlakyInfAttempts(t *testing.T) {
	testAsyncLBFlakyInfAttempts(t, modetest.SingleEvent(testEvent))
}

func TestAsyncLBMultiFlakyInfAttempts(t *testing.T) {
	testAsyncLBFlakyInfAttempts(t, modetest.MultiEvent(10, testEvent))
}

func testAsyncLBFlakyInfAttempts2(t *testing.T, events []modetest.EventInfo) {
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"*"})
	}

	var collected [][]outputs.Data
	err := errors.New("flaky")
	mode, _ := NewAsync(
		modetest.AsyncClients(2, &modetest.MockClient{
			CBAsyncPublish: modetest.AsyncPublishCollectAfterFailStartWith(
				50, err, &collected),
		}),
		0,
		1*time.Nanosecond,
		1*time.Millisecond,
		4*time.Millisecond,
	)
	modetest.TestMode(t, mode, testNoOpts, events, modetest.Signals(true), &collected)
}

func TestAsyncLBFlakyInfAttempts2(t *testing.T) {
	testAsyncLBFlakyInfAttempts2(t, modetest.SingleEvent(testEvent))
}

func TestAsyncLBMultiFlakyInfAttempts2(t *testing.T) {
	testAsyncLBFlakyInfAttempts2(t, modetest.MultiEvent(10, testEvent))
}

func testAsyncLBFlakyGuaranteed(t *testing.T, events []modetest.EventInfo) {
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"*"})
	}

	var collected [][]outputs.Data
	err := errors.New("flaky")
	tmpl := &modetest.MockClient{
		Connected:      true,
		CBAsyncPublish: modetest.AsyncPublishCollectAfterFailStartWith(50, err, &collected),
	}

	mode, _ := NewAsync(
		modetest.AsyncClients(2, tmpl),
		3,
		1*time.Nanosecond,
		1*time.Millisecond,
		4*time.Millisecond,
	)
	modetest.TestMode(t, mode, testGuaranteed, events, modetest.Signals(true), &collected)
}

func TestAsyncLBFlakyGuaranteed(t *testing.T) {
	testAsyncLBFlakyGuaranteed(t, modetest.SingleEvent(testEvent))
}

func TestAsyncLBMultiFlakyGuaranteed(t *testing.T) {
	testAsyncLBFlakyGuaranteed(t, modetest.MultiEvent(10, testEvent))
}

func testAsyncLBFlakyGuaranteed2(t *testing.T, events []modetest.EventInfo) {
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"*"})
	}

	var collected [][]outputs.Data
	err := errors.New("flaky")
	tmpl := &modetest.MockClient{
		Connected:      true,
		CBAsyncPublish: modetest.AsyncPublishCollectAfterFailStartWith(50, err, &collected),
	}
	mode, _ := NewAsync(
		modetest.AsyncClients(2, tmpl),
		3,
		1*time.Nanosecond,
		1*time.Millisecond,
		4*time.Millisecond,
	)
	modetest.TestMode(t, mode, testGuaranteed, events, modetest.Signals(true), &collected)
}

func TestAsyncLBFlakyGuaranteed2(t *testing.T) {
	testAsyncLBFlakyGuaranteed2(t, modetest.SingleEvent(testEvent))
}

func TestAsyncLBMultiFlakyGuaranteed2(t *testing.T) {
	testAsyncLBFlakyGuaranteed2(t, modetest.MultiEvent(10, testEvent))
}
