package mode

import (
	"errors"
	"testing"
	"time"

	"github.com/elastic/libbeat/common"
)

func testSingleSendOneEvent(t *testing.T, events []eventInfo) {
	var collected [][]common.MapStr
	mode, _ := NewSingleConnectionMode(
		&mockClient{
			connected: true,
			close:     closeOK,
			connect:   connectOK,
			publish:   collectPublish(&collected),
		},
		3,
		0,
		100*time.Millisecond,
		1*time.Second,
	)
	testMode(t, mode, events, signals(true), &collected)
}

func TestSingleSendOneEvent(t *testing.T) {
	testSingleSendOneEvent(t, singleEvent(testEvent))
}

func TestSingleSendMultiple(t *testing.T) {
	testSingleSendOneEvent(t, multiEvent(10, testEvent))
}

func testSingleConnectFailConnectAndSend(t *testing.T, events []eventInfo) {
	var collected [][]common.MapStr
	errFail := errors.New("fail connect")
	mode, _ := NewSingleConnectionMode(
		&mockClient{
			connected: false,
			close:     closeOK,
			connect:   failConnect(3, errFail),
			publish:   collectPublish(&collected),
		},
		3,
		0,
		100*time.Millisecond,
		1*time.Second,
	)
	testMode(t, mode, events, signals(true), &collected)
}

func TestSingleConnectFailConnectAndSend(t *testing.T) {
	testSingleConnectFailConnectAndSend(t, singleEvent(testEvent))
}

func TestSingleConnectFailConnectAndSendMultiple(t *testing.T) {
	testSingleConnectFailConnectAndSend(t, multiEvent(10, testEvent))
}

func testSingleConnectionFail(t *testing.T, events []eventInfo) {
	var collected [][]common.MapStr
	errFail := errors.New("fail connect")
	mode, _ := NewSingleConnectionMode(
		&mockClient{
			connected: false,
			close:     closeOK,
			connect:   alwaysFailConnect(errFail),
			publish:   collectPublish(&collected),
		},
		3,
		0,
		100*time.Millisecond,
		1*time.Second,
	)
	testMode(t, mode, events, signals(false), &collected)
}

func TestSingleConnectionFail(t *testing.T) {
	testSingleConnectionFail(t, singleEvent(testEvent))
}

func TestSingleConnectionFailMulti(t *testing.T) {
	testSingleConnectionFail(t, multiEvent(10, testEvent))
}

func testSingleSendFlaky(t *testing.T, events []eventInfo) {
	var collected [][]common.MapStr
	mode, _ := NewSingleConnectionMode(
		&mockClient{
			connected: false,
			close:     closeOK,
			connect:   connectOK,
			publish:   publishFailStart(2, collectPublish(&collected)),
		},
		3,
		0,
		100*time.Millisecond,
		1*time.Second,
	)
	testMode(t, mode, singleEvent(testEvent), signals(true), &collected)
}

func TestSingleSendFlaky(t *testing.T) {
	testSingleSendFlaky(t, singleEvent(testEvent))
}

func TestSingleSendMultiFlaky(t *testing.T) {
	testSingleSendFlaky(t, multiEvent(10, testEvent))
}
