package mode

import (
	"errors"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common"
)

func testFailoverSend(t *testing.T, events []eventInfo) {
	var collected [][]common.MapStr
	mode, _ := NewFailOverConnectionMode(
		[]ProtocolClient{
			&mockClient{
				connected: false,
				close:     closeOK,
				connect:   connectOK,
				publish:   collectPublish(&collected),
			},
			&mockClient{
				connected: false,
				close:     closeOK,
				connect:   connectOK,
				publish:   collectPublish(&collected),
			},
		},
		3,
		0,
		1*time.Millisecond,
		4*time.Millisecond,
	)
	testMode(t, mode, testNoOpts, events, signals(true), &collected)
}

func TestFailoverSingleSendOne(t *testing.T) {
	testFailoverSend(t, singleEvent(testEvent))
}

func TestFailoverSendMultiple(t *testing.T) {
	testFailoverSend(t, multiEvent(10, testEvent))
}

func testFailoverConnectFailAndSend(t *testing.T, events []eventInfo) {
	errFail := errors.New("fail connect")
	var collected [][]common.MapStr
	mode, _ := NewFailOverConnectionMode(
		[]ProtocolClient{
			&mockClient{
				connected: false,
				close:     closeOK,
				connect:   failConnect(3, errFail),
				publish:   publishTimeoutEvery(1, collectPublish(&collected)),
			},
			&mockClient{
				connected: false,
				close:     closeOK,
				connect:   failConnect(2, errFail),
				publish:   publishTimeoutEvery(2, collectPublish(&collected)),
			},
		},
		3,
		1*time.Millisecond,
		1*time.Millisecond,
		4*time.Millisecond,
	)
	testMode(t, mode, testNoOpts, events, signals(true), &collected)
}

func TestFailoverConnectFailAndSend(t *testing.T) {
	testFailoverConnectFailAndSend(t, singleEvent(testEvent))
}

func TestFailoverConnectFailConnectAndSendMultiple(t *testing.T) {
	testFailoverConnectFailAndSend(t, multiEvent(10, testEvent))
}

func testFailoverConnectionFail(t *testing.T, events []eventInfo) {
	errFail := errors.New("fail connect")
	var collected [][]common.MapStr
	mode, _ := NewFailOverConnectionMode(
		[]ProtocolClient{
			&mockClient{
				connected: false,
				close:     closeOK,
				connect:   alwaysFailConnect(errFail),
				publish:   publishTimeoutEvery(1, collectPublish(&collected)),
			},
			&mockClient{
				connected: false,
				close:     closeOK,
				connect:   alwaysFailConnect(errFail),
				publish:   publishTimeoutEvery(2, collectPublish(&collected)),
			},
		},
		3,
		1*time.Millisecond,
		1*time.Millisecond,
		4*time.Millisecond,
	)
	testMode(t, mode, testNoOpts, events, signals(false), &collected)
}

func TestFailoverConnectionFail(t *testing.T) {
	testFailoverConnectionFail(t, singleEvent(testEvent))
}

func TestFailoverConnectionFailMulti(t *testing.T) {
	testFailoverConnectionFail(t, multiEvent(10, testEvent))
}

func testFailoverSendFlaky(t *testing.T, events []eventInfo) {
	var collected [][]common.MapStr
	mode, _ := NewFailOverConnectionMode(
		[]ProtocolClient{
			&mockClient{
				connected: false,
				close:     closeOK,
				connect:   connectOK,
				publish:   publishFailStart(1, collectPublish(&collected)),
			},
			&mockClient{
				connected: false,
				close:     closeOK,
				connect:   connectOK,
				publish:   publishFailStart(1, collectPublish(&collected)),
			},
		},
		3,
		1*time.Millisecond,
		1*time.Millisecond,
		4*time.Millisecond,
	)
	testMode(t, mode, testNoOpts, events, signals(true), &collected)
}

func TestFailoverSendFlaky(t *testing.T) {
	testFailoverSendFlaky(t, singleEvent(testEvent))
}

func TestFailoverSendMultiFlaky(t *testing.T) {
	testFailoverSendFlaky(t, multiEvent(10, testEvent))
}

func testFailoverSendFlakyFail(t *testing.T, events []eventInfo) {
	var collected [][]common.MapStr
	mode, _ := NewFailOverConnectionMode(
		[]ProtocolClient{
			&mockClient{
				connected: false,
				close:     closeOK,
				connect:   connectOK,
				publish:   publishFailStart(2, collectPublish(&collected)),
			},
			&mockClient{
				connected: false,
				close:     closeOK,
				connect:   connectOK,
				publish:   publishFailStart(2, collectPublish(&collected)),
			},
		},
		3,
		1*time.Millisecond,
		1*time.Millisecond,
		4*time.Millisecond,
	)
	testMode(t, mode, testNoOpts, events, signals(false), &collected)
}

func TestFailoverSendFlakyFail(t *testing.T) {
	testFailoverSendFlakyFail(t, singleEvent(testEvent))
}

func TestFailoverSendMultiFlakyFail(t *testing.T) {
	testFailoverSendFlakyFail(t, multiEvent(10, testEvent))
}

func testFailoverSendFlakyInfAttempts(t *testing.T, events []eventInfo) {
	var collected [][]common.MapStr
	mode, _ := NewFailOverConnectionMode(
		[]ProtocolClient{
			&mockClient{
				connected: false,
				close:     closeOK,
				connect:   connectOK,
				publish:   publishFailStart(50, collectPublish(&collected)),
			},
			&mockClient{
				connected: false,
				close:     closeOK,
				connect:   connectOK,
				publish:   publishFailStart(50, collectPublish(&collected)),
			},
		},
		0,
		1*time.Millisecond,
		1*time.Millisecond,
		4*time.Millisecond,
	)
	testMode(t, mode, testNoOpts, events, signals(true), &collected)
}

func TestFailoverSendFlakyInfAttempts(t *testing.T) {
	testFailoverSendFlakyInfAttempts(t, singleEvent(testEvent))
}

func TestFailoverSendMultiFlakyInfAttempts(t *testing.T) {
	testFailoverSendFlakyInfAttempts(t, multiEvent(10, testEvent))
}

func testFailoverSendFlakyGuaranteed(t *testing.T, events []eventInfo) {
	var collected [][]common.MapStr
	mode, _ := NewFailOverConnectionMode(
		[]ProtocolClient{
			&mockClient{
				connected: false,
				close:     closeOK,
				connect:   connectOK,
				publish:   publishFailStart(50, collectPublish(&collected)),
			},
			&mockClient{
				connected: false,
				close:     closeOK,
				connect:   connectOK,
				publish:   publishFailStart(50, collectPublish(&collected)),
			},
		},
		3,
		1*time.Millisecond,
		1*time.Millisecond,
		4*time.Millisecond,
	)
	testMode(t, mode, testGuaranteed, events, signals(true), &collected)
}

func TestFailoverSendFlakyGuaranteed(t *testing.T) {
	testFailoverSendFlakyGuaranteed(t, singleEvent(testEvent))
}

func TestFailoverSendMultiFlakyGuaranteed(t *testing.T) {
	testFailoverSendFlakyGuaranteed(t, multiEvent(10, testEvent))
}
