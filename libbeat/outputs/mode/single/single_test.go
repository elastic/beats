// +build !integration

package single

import (
	"errors"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/mode/modetest"
)

var (
	testNoOpts     = outputs.Options{}
	testGuaranteed = outputs.Options{Guaranteed: true}

	testEvent = common.MapStr{
		"msg": "hello world",
	}
)

func enableLogging(selectors []string) {
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, selectors)
	}
}

func testSingleSendOneEvent(t *testing.T, events []modetest.EventInfo) {
	var collected [][]outputs.Data
	mode, _ := New(
		modetest.NewMockClient(&modetest.MockClient{
			Connected: true,
			CBPublish: modetest.PublishCollect(&collected),
		}),
		3,
		1*time.Millisecond,
		1*time.Millisecond,
		10*time.Millisecond,
	)
	modetest.TestMode(t, mode, testNoOpts, events, modetest.Signals(true), &collected)
}

func TestSingleSendOneEvent(t *testing.T) {
	testSingleSendOneEvent(t, modetest.SingleEvent(testEvent))
}

func TestSingleSendMultiple(t *testing.T) {
	testSingleSendOneEvent(t, modetest.MultiEvent(10, testEvent))
}

func testSingleConnectFailConnectAndSend(t *testing.T, events []modetest.EventInfo) {
	var collected [][]outputs.Data
	errFail := errors.New("fail connect")
	mode, _ := New(
		modetest.NewMockClient(&modetest.MockClient{
			Connected: false,
			CBConnect: modetest.ConnectFailN(2, errFail),
			CBPublish: modetest.PublishCollect(&collected),
		}),
		3,
		1*time.Millisecond,
		1*time.Millisecond,
		10*time.Millisecond,
	)
	modetest.TestMode(t, mode, testNoOpts, events, modetest.Signals(true), &collected)
}

func TestSingleConnectFailConnectAndSend(t *testing.T) {
	testSingleConnectFailConnectAndSend(t, modetest.SingleEvent(testEvent))
}

func TestSingleConnectFailConnectAndSendMultiple(t *testing.T) {
	testSingleConnectFailConnectAndSend(t, modetest.MultiEvent(10, testEvent))
}

func testSingleConnectionFail(t *testing.T, events []modetest.EventInfo) {
	var collected [][]outputs.Data
	errFail := errors.New("fail connect")
	mode, _ := New(
		modetest.NewMockClient(&modetest.MockClient{
			Connected: false,
			CBConnect: modetest.ConnectFail(errFail),
			CBPublish: modetest.PublishCollect(&collected),
		}),
		3,
		1*time.Millisecond,
		1*time.Millisecond,
		10*time.Millisecond,
	)
	modetest.TestMode(t, mode, testNoOpts, events, modetest.Signals(false), &collected)
}

func TestSingleConnectionFail(t *testing.T) {
	testSingleConnectionFail(t, modetest.SingleEvent(testEvent))
}

func TestSingleConnectionFailMulti(t *testing.T) {
	testSingleConnectionFail(t, modetest.MultiEvent(10, testEvent))
}

func testSingleSendFlaky(t *testing.T, events []modetest.EventInfo) {
	var collected [][]outputs.Data
	mode, _ := New(
		modetest.NewMockClient(&modetest.MockClient{
			CBPublish: modetest.PublishCollectAfterFailStart(2, &collected),
		}),
		3,
		1*time.Millisecond,
		1*time.Millisecond,
		10*time.Millisecond,
	)
	modetest.TestMode(t, mode, testNoOpts, events, modetest.Signals(true), &collected)
}

func TestSingleSendFlaky(t *testing.T) {
	testSingleSendFlaky(t, modetest.SingleEvent(testEvent))
}

func TestSingleSendMultiFlaky(t *testing.T) {
	testSingleSendFlaky(t, modetest.MultiEvent(10, testEvent))
}

func testSingleSendFlakyFail(t *testing.T, events []modetest.EventInfo) {
	var collected [][]outputs.Data
	mode, _ := New(
		modetest.NewMockClient(&modetest.MockClient{
			CBPublish: modetest.PublishCollectAfterFailStart(3, &collected),
		}),
		3,
		1*time.Millisecond,
		1*time.Millisecond,
		10*time.Millisecond,
	)
	modetest.TestMode(t, mode, testNoOpts, events, modetest.Signals(false), &collected)
}

func TestSingleSendFlakyFail(t *testing.T) {
	testSingleSendFlakyFail(t, modetest.SingleEvent(testEvent))
}

func TestSingleSendMultiFlakyFail(t *testing.T) {
	testSingleSendFlakyFail(t, modetest.MultiEvent(10, testEvent))
}

func testSingleSendFlakyInfAttempts(t *testing.T, events []modetest.EventInfo) {
	enableLogging([]string{"*"})

	var collected [][]outputs.Data
	mode, _ := New(
		modetest.NewMockClient(&modetest.MockClient{
			CBPublish: modetest.PublishCollectAfterFailStart(25, &collected),
		}),
		0, // infinite number of send attempts
		1*time.Millisecond,
		1*time.Millisecond,
		10*time.Millisecond,
	)
	modetest.TestMode(t, mode, testNoOpts, events, modetest.Signals(true), &collected)
}

func TestSingleSendFlakyInfAttempts(t *testing.T) {
	testSingleSendFlakyInfAttempts(t, modetest.SingleEvent(testEvent))
}

func TestSingleSendMultiFlakyInfAttempts(t *testing.T) {
	testSingleSendFlakyInfAttempts(t, modetest.MultiEvent(10, testEvent))
}

func testSingleSendFlakyGuaranteed(t *testing.T, events []modetest.EventInfo) {
	var collected [][]outputs.Data
	mode, _ := New(
		modetest.NewMockClient(&modetest.MockClient{
			CBPublish: modetest.PublishCollectAfterFailStart(25, &collected),
		}),
		3,
		1*time.Millisecond,
		1*time.Millisecond,
		10*time.Millisecond,
	)
	modetest.TestMode(t, mode, testGuaranteed, events, modetest.Signals(true), &collected)
}

func TestSingleSendFlakyGuaranteed(t *testing.T) {
	testSingleSendFlakyGuaranteed(t, modetest.SingleEvent(testEvent))
}

func TestSingleSendMultiFlakyGuaranteed(t *testing.T) {
	testSingleSendFlakyGuaranteed(t, modetest.MultiEvent(10, testEvent))
}
