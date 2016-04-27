// +build !integration

package mode

import (
	"errors"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common"
)

func testSingleSendOneEvent(t *testing.T, events []eventInfo) {
	var collected [][]common.MapStr
	mode, _ := NewConnectionMode(
		[]ProtocolClient{
			&mockClient{
				connected: true,
				close:     closeOK,
				connect:   connectOK,
				publish:   collectPublish(&collected),
			},
		},
		false,
		3,
		1*time.Millisecond,
		1*time.Millisecond,
		10*time.Millisecond,
	)
	testMode(t, mode, testNoOpts, events, signals(true), &collected)
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
	mode, _ := NewConnectionMode(
		[]ProtocolClient{
			&mockClient{
				connected: false,
				close:     closeOK,
				connect:   failConnect(3, errFail), // 3 fails
				publish:   collectPublish(&collected),
			},
		},
		false,
		3,
		1*time.Millisecond,
		1*time.Millisecond,
		10*time.Millisecond,
	)
	testMode(t, mode, testNoOpts, events, signals(true), &collected)
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
	mode, _ := NewConnectionMode(
		[]ProtocolClient{
			&mockClient{
				connected: false,
				close:     closeOK,
				connect:   alwaysFailConnect(errFail),
				publish:   collectPublish(&collected),
			},
		},
		false,
		3,
		1*time.Millisecond,
		1*time.Millisecond,
		10*time.Millisecond,
	)
	testMode(t, mode, testNoOpts, events, signals(false), &collected)
}

func TestSingleConnectionFail(t *testing.T) {
	testSingleConnectionFail(t, singleEvent(testEvent))
}

func TestSingleConnectionFailMulti(t *testing.T) {
	testSingleConnectionFail(t, multiEvent(10, testEvent))
}

func testSingleSendFlaky(t *testing.T, events []eventInfo) {
	var collected [][]common.MapStr
	mode, _ := NewConnectionMode(
		[]ProtocolClient{
			&mockClient{
				connected: false,
				close:     closeOK,
				connect:   connectOK,
				publish:   publishFailStart(2, collectPublish(&collected)),
			},
		},
		false,
		3,
		1*time.Millisecond,
		1*time.Millisecond,
		10*time.Millisecond,
	)
	testMode(t, mode, testNoOpts, events, signals(true), &collected)
}

func TestSingleSendFlaky(t *testing.T) {
	testSingleSendFlaky(t, singleEvent(testEvent))
}

func TestSingleSendMultiFlaky(t *testing.T) {
	testSingleSendFlaky(t, multiEvent(10, testEvent))
}

func testSingleSendFlakyFail(t *testing.T, events []eventInfo) {
	var collected [][]common.MapStr
	mode, _ := NewConnectionMode(
		[]ProtocolClient{
			&mockClient{
				connected: false,
				close:     closeOK,
				connect:   connectOK,
				publish:   publishFailStart(3, collectPublish(&collected)),
			},
		},
		false,
		3,
		1*time.Millisecond,
		1*time.Millisecond,
		10*time.Millisecond,
	)
	testMode(t, mode, testNoOpts, events, signals(false), &collected)
}

func TestSingleSendFlakyFail(t *testing.T) {
	testSingleSendFlakyFail(t, singleEvent(testEvent))
}

func TestSingleSendMultiFlakyFail(t *testing.T) {
	testSingleSendFlakyFail(t, multiEvent(10, testEvent))
}

func testSingleSendFlakyInfAttempts(t *testing.T, events []eventInfo) {
	enableLogging([]string{"*"})

	var collected [][]common.MapStr
	mode, _ := NewConnectionMode(
		[]ProtocolClient{
			&mockClient{
				connected: false,
				close:     closeOK,
				connect:   connectOK,
				publish:   publishFailStart(25, collectPublish(&collected)),
			},
		},
		false,
		0, // infinite number of send attempts
		1*time.Millisecond,
		1*time.Millisecond,
		10*time.Millisecond,
	)
	testMode(t, mode, testNoOpts, events, signals(true), &collected)
}

func TestSingleSendFlakyInfAttempts(t *testing.T) {
	testSingleSendFlakyInfAttempts(t, singleEvent(testEvent))
}

func TestSingleSendMultiFlakyInfAttempts(t *testing.T) {
	testSingleSendFlakyInfAttempts(t, multiEvent(10, testEvent))
}

func testSingleSendFlakyGuaranteed(t *testing.T, events []eventInfo) {
	var collected [][]common.MapStr
	mode, _ := NewConnectionMode(
		[]ProtocolClient{
			&mockClient{
				connected: false,
				close:     closeOK,
				connect:   connectOK,
				publish:   publishFailStart(25, collectPublish(&collected)),
			},
		},
		false,
		3,
		1*time.Millisecond,
		1*time.Millisecond,
		10*time.Millisecond,
	)
	testMode(t, mode, testGuaranteed, events, signals(true), &collected)
}

func TestSingleSendFlakyGuaranteed(t *testing.T) {
	testSingleSendFlakyGuaranteed(t, singleEvent(testEvent))
}

func TestSingleSendMultiFlakyGuaranteed(t *testing.T) {
	testSingleSendFlakyGuaranteed(t, multiEvent(10, testEvent))
}
