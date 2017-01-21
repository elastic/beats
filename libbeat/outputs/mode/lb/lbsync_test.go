// +build !integration

package lb

import (
	"errors"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/mode"
	"github.com/elastic/beats/libbeat/outputs/mode/modetest"
)

func TestLoadBalancerStartStop(t *testing.T) {
	enableLogging([]string{"*"})

	mode, _ := NewSync(
		modetest.SyncClients(1, &modetest.MockClient{
			Connected: false,
		}),
		1,
		1*time.Millisecond,
		1*time.Millisecond,
		10*time.Millisecond,
	)
	modetest.TestMode(t, mode, testNoOpts, nil, nil, nil)
}

func testLoadBalancerFailSendWithoutActiveConnections(
	t *testing.T,
	events []modetest.EventInfo,
) {
	errFail := errors.New("fail connect")
	mode, _ := NewSync(
		modetest.SyncClients(2, &modetest.MockClient{
			Connected: false,
			CBConnect: modetest.ConnectFail(errFail),
		}),
		2,
		1*time.Millisecond,
		1*time.Millisecond,
		10*time.Millisecond,
	)
	modetest.TestMode(t, mode, testNoOpts, events, modetest.Signals(false), nil)
}

func TestLoadBalancerFailSendWithoutActiveConnections(t *testing.T) {
	testLoadBalancerFailSendWithoutActiveConnections(t, modetest.SingleEvent(testEvent))
}

func TestLoadBalancerFailSendMultWithoutActiveConnections(t *testing.T) {
	testLoadBalancerFailSendWithoutActiveConnections(t, modetest.MultiEvent(2, testEvent))
}

func testLoadBalancerOKSend(t *testing.T, events []modetest.EventInfo) {
	var collected [][]outputs.Data
	mode, _ := NewSync(
		modetest.SyncClients(1, &modetest.MockClient{
			Connected: false,
			CBPublish: modetest.PublishCollect(&collected),
		}),
		2,
		1*time.Millisecond,
		1*time.Millisecond,
		10*time.Millisecond,
	)
	modetest.TestMode(t, mode, testNoOpts, events, modetest.Signals(true), &collected)
}

func TestLoadBalancerOKSend(t *testing.T) {
	testLoadBalancerOKSend(t, modetest.SingleEvent(testEvent))
}

func TestLoadBalancerOKSendMult(t *testing.T) {
	testLoadBalancerOKSend(t, modetest.MultiEvent(10, testEvent))
}

func testLoadBalancerFlakyConnectionOkSend(t *testing.T, events []modetest.EventInfo) {
	var collected [][]outputs.Data
	mode, _ := NewSync(
		modetest.SyncClients(2, &modetest.MockClient{
			Connected: true,
			CBPublish: modetest.PublishCollectAfterFailStart(1, &collected),
		}),
		3,
		1*time.Millisecond,
		1*time.Millisecond,
		10*time.Millisecond,
	)
	modetest.TestMode(t, mode, testNoOpts, events, modetest.Signals(true), &collected)
}

func TestLoadBalancerFlakyConnectionOkSend(t *testing.T) {
	testLoadBalancerFlakyConnectionOkSend(t, modetest.SingleEvent(testEvent))
}

func TestLoadBalancerFlakyConnectionOkSendMult(t *testing.T) {
	testLoadBalancerFlakyConnectionOkSend(t, modetest.MultiEvent(10, testEvent))
}

func testLoadBalancerFlakyFail(t *testing.T, events []modetest.EventInfo) {
	var collected [][]outputs.Data
	mode, _ := NewSync(
		modetest.SyncClients(2, &modetest.MockClient{
			Connected: true,
			CBPublish: modetest.PublishCollectAfterFailStart(3, &collected),
		}),
		3,
		1*time.Millisecond,
		1*time.Millisecond,
		10*time.Millisecond,
	)
	modetest.TestMode(t, mode, testNoOpts, events, modetest.Signals(false), &collected)
}

func TestLoadBalancerFlakyFail(t *testing.T) {
	testLoadBalancerFlakyFail(t, modetest.SingleEvent(testEvent))
}

func TestLoadBalancerMultiFlakyFail(t *testing.T) {
	testLoadBalancerFlakyFail(t, modetest.MultiEvent(10, testEvent))
}

func testLoadBalancerTemporayFailure(t *testing.T, events []modetest.EventInfo) {
	var collected [][]outputs.Data
	err := mode.ErrTempBulkFailure
	mode, _ := NewSync(
		modetest.SyncClients(1, &modetest.MockClient{
			Connected: true,
			CBPublish: modetest.PublishCollectAfterFailStartWith(1, err, &collected),
		}),
		3,
		1*time.Millisecond,
		1*time.Millisecond,
		10*time.Millisecond,
	)
	modetest.TestMode(t, mode, testNoOpts, events, modetest.Signals(true), &collected)
}

func TestLoadBalancerTemporayFailure(t *testing.T) {
	testLoadBalancerTemporayFailure(t, modetest.SingleEvent(testEvent))
}

func TestLoadBalancerTemporayFailureMutlEvents(t *testing.T) {
	testLoadBalancerTemporayFailure(t, modetest.MultiEvent(10, testEvent))
}

func testLoadBalancerTempFlakyFail(t *testing.T, events []modetest.EventInfo) {
	var collected [][]outputs.Data
	err := mode.ErrTempBulkFailure
	mode, _ := NewSync(
		modetest.SyncClients(2, &modetest.MockClient{
			Connected: true,
			CBPublish: modetest.PublishCollectAfterFailStartWith(3, err, &collected),
		}),
		3,
		1*time.Millisecond,
		1*time.Millisecond,
		10*time.Millisecond,
	)
	modetest.TestMode(t, mode, testNoOpts, events, modetest.Signals(false), &collected)
}

func TestLoadBalancerTempFlakyFail(t *testing.T) {
	testLoadBalancerTempFlakyFail(t, modetest.SingleEvent(testEvent))
}

func TestLoadBalancerMultiTempFlakyFail(t *testing.T) {
	testLoadBalancerTempFlakyFail(t, modetest.MultiEvent(10, testEvent))
}

func testLoadBalancerFlakyInfAttempts(t *testing.T, events []modetest.EventInfo) {
	var collected [][]outputs.Data
	mode, _ := NewSync(
		modetest.SyncClients(2, &modetest.MockClient{
			Connected: true,
			CBPublish: modetest.PublishCollectAfterFailStart(25, &collected),
		}),
		0,
		1*time.Millisecond,
		1*time.Millisecond,
		10*time.Millisecond,
	)
	modetest.TestMode(t, mode, testNoOpts, events, modetest.Signals(true), &collected)
}

func TestLoadBalancerFlakyInfAttempts(t *testing.T) {
	testLoadBalancerFlakyInfAttempts(t, modetest.SingleEvent(testEvent))
}

func TestLoadBalancerMultiFlakyInfAttempts(t *testing.T) {
	testLoadBalancerFlakyInfAttempts(t, modetest.MultiEvent(10, testEvent))
}

func testLoadBalancerTempFlakyInfAttempts(t *testing.T, events []modetest.EventInfo) {
	var collected [][]outputs.Data
	err := mode.ErrTempBulkFailure
	mode, _ := NewSync(
		modetest.SyncClients(2, &modetest.MockClient{
			Connected: true,
			CBPublish: modetest.PublishCollectAfterFailStartWith(25, err, &collected),
		}),
		0,
		1*time.Millisecond,
		1*time.Millisecond,
		10*time.Millisecond,
	)
	modetest.TestMode(t, mode, testNoOpts, events, modetest.Signals(true), &collected)
}

func TestLoadBalancerTempFlakyInfAttempts(t *testing.T) {
	testLoadBalancerTempFlakyInfAttempts(t, modetest.SingleEvent(testEvent))
}

func TestLoadBalancerMultiTempFlakyInfAttempts(t *testing.T) {
	testLoadBalancerTempFlakyInfAttempts(t, modetest.MultiEvent(10, testEvent))
}

func testLoadBalancerFlakyGuaranteed(t *testing.T, events []modetest.EventInfo) {
	var collected [][]outputs.Data
	mode, _ := NewSync(
		modetest.SyncClients(2, &modetest.MockClient{
			Connected: true,
			CBPublish: modetest.PublishCollectAfterFailStart(25, &collected),
		}),
		3,
		1*time.Millisecond,
		1*time.Millisecond,
		10*time.Millisecond,
	)
	modetest.TestMode(t, mode, testGuaranteed, events, modetest.Signals(true), &collected)
}

func TestLoadBalancerFlakyGuaranteed(t *testing.T) {
	testLoadBalancerFlakyGuaranteed(t, modetest.SingleEvent(testEvent))
}

func TestLoadBalancerMultiFlakyGuaranteed(t *testing.T) {
	testLoadBalancerFlakyGuaranteed(t, modetest.MultiEvent(10, testEvent))
}

func testLoadBalancerTempFlakyGuaranteed(t *testing.T, events []modetest.EventInfo) {
	var collected [][]outputs.Data
	err := mode.ErrTempBulkFailure
	mode, _ := NewSync(
		modetest.SyncClients(2, &modetest.MockClient{
			Connected: true,
			CBPublish: modetest.PublishCollectAfterFailStartWith(25, err, &collected),
		}),
		3,
		1*time.Millisecond,
		1*time.Millisecond,
		10*time.Millisecond,
	)
	modetest.TestMode(t, mode, testGuaranteed, events, modetest.Signals(true), &collected)
}

func TestLoadBalancerTempFlakyGuaranteed(t *testing.T) {
	testLoadBalancerTempFlakyGuaranteed(t, modetest.SingleEvent(testEvent))
}

func TestLoadBalancerMultiTempFlakyGuaranteed(t *testing.T) {
	testLoadBalancerTempFlakyGuaranteed(t, modetest.MultiEvent(10, testEvent))
}
