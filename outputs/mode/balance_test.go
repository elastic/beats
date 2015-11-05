package mode

import (
	"errors"
	"testing"
	"time"

	"github.com/elastic/libbeat/common"
)

func TestLoadBalancerStartStop(t *testing.T) {
	mode, _ := NewLoadBalancerMode(
		[]ProtocolClient{
			&mockClient{
				connected: false,
				close:     closeOK,
				connect:   connectOK,
			},
		},
		1,
		100*time.Millisecond,
		100*time.Millisecond,
		1*time.Second,
	)
	testMode(t, mode, nil, nil, nil)
}

func testLoadBalancerFailSendWithoutActiveConnections(
	t *testing.T,
	events []eventInfo,
) {
	errFail := errors.New("fail connect")
	mode, _ := NewLoadBalancerMode(
		[]ProtocolClient{
			&mockClient{
				connected: false,
				close:     closeOK,
				connect:   alwaysFailConnect(errFail),
			},
			&mockClient{
				connected: false,
				close:     closeOK,
				connect:   alwaysFailConnect(errFail),
			},
		},
		2,
		100*time.Millisecond,
		100*time.Millisecond,
		1*time.Second,
	)
	testMode(t, mode, events, signals(false), nil)
}

func TestLoadBalancerFailSendWithoutActiveConnections(t *testing.T) {
	testLoadBalancerFailSendWithoutActiveConnections(t, singleEvent(testEvent))
}

func TestLoadBalancerFailSendMultWithoutActiveConnections(t *testing.T) {
	testLoadBalancerFailSendWithoutActiveConnections(t, multiEvent(2, testEvent))
}

func testLoadBalancerOKSend(t *testing.T, events []eventInfo) {
	var collected [][]common.MapStr
	mode, _ := NewLoadBalancerMode(
		[]ProtocolClient{
			&mockClient{
				connected: false,
				close:     closeOK,
				connect:   connectOK,
				publish:   collectPublish(&collected),
			},
		},
		2,
		100*time.Millisecond,
		100*time.Millisecond,
		1*time.Second,
	)
	testMode(t, mode, events, signals(true), &collected)
}

func TestLoadBalancerOKSend(t *testing.T) {
	testLoadBalancerOKSend(t, singleEvent(testEvent))
}

func TestLoadBalancerOKSendMult(t *testing.T) {
	testLoadBalancerOKSend(t, multiEvent(10, testEvent))
}

func testLoadBalancerFlakyConnectionOkSend(t *testing.T, events []eventInfo) {
	var collected [][]common.MapStr
	mode, _ := NewLoadBalancerMode(
		[]ProtocolClient{
			&mockClient{
				connected: true,
				close:     closeOK,
				connect:   connectOK,
				publish:   publishFailStart(1, collectPublish(&collected)),
			},
			&mockClient{
				connected: true,
				close:     closeOK,
				connect:   connectOK,
				publish:   publishFailStart(1, collectPublish(&collected)),
			},
		},
		3,
		100*time.Millisecond,
		100*time.Millisecond,
		1*time.Second,
	)
	testMode(t, mode, events, signals(true), &collected)
}

func TestLoadBalancerFlakyConnectionOkSend(t *testing.T) {
	testLoadBalancerFlakyConnectionOkSend(t, singleEvent(testEvent))
}

func TestLoadBalancerFlakyConnectionOkSendMult(t *testing.T) {
	testLoadBalancerFlakyConnectionOkSend(t, multiEvent(10, testEvent))
}

func testLoadBalancerFlakyFail(t *testing.T, events []eventInfo) {
	var collected [][]common.MapStr
	mode, _ := NewLoadBalancerMode(
		[]ProtocolClient{
			&mockClient{
				connected: true,
				close:     closeOK,
				connect:   connectOK,
				publish:   publishFailStart(3, collectPublish(&collected)),
			},
			&mockClient{
				connected: true,
				close:     closeOK,
				connect:   connectOK,
				publish:   publishFailStart(3, collectPublish(&collected)),
			},
		},
		3,
		100*time.Millisecond,
		100*time.Millisecond,
		1*time.Second,
	)
	testMode(t, mode, events, signals(false), &collected)
}

func TestLoadBalancerFlakyFail(t *testing.T) {
	testLoadBalancerFlakyFail(t, singleEvent(testEvent))
}

func TestLoadBalancerMultiFlakyFail(t *testing.T) {
	testLoadBalancerFlakyFail(t, multiEvent(10, testEvent))
}

func testLoadBalancerTemporayFailure(t *testing.T, events []eventInfo) {
	var collected [][]common.MapStr
	mode, _ := NewLoadBalancerMode(
		[]ProtocolClient{
			&mockClient{
				connected: true,
				close:     closeOK,
				connect:   connectOK,
				publish:   publishFailWith(1, ErrTempBulkFailure, collectPublish(&collected)),
			},
		},
		3,
		100*time.Millisecond,
		100*time.Millisecond,
		1*time.Second,
	)
	testMode(t, mode, events, signals(true), &collected)
}

func TestLoadBalancerTemporayFailure(t *testing.T) {
	testLoadBalancerTemporayFailure(t, singleEvent(testEvent))
}

func TestLoadBalancerTemporayFailureMutlEvents(t *testing.T) {
	testLoadBalancerTemporayFailure(t, multiEvent(10, testEvent))
}

func testLoadBalancerTempFlakyFail(t *testing.T, events []eventInfo) {
	var collected [][]common.MapStr
	mode, _ := NewLoadBalancerMode(
		[]ProtocolClient{
			&mockClient{
				connected: true,
				close:     closeOK,
				connect:   connectOK,
				publish:   publishFailWith(3, ErrTempBulkFailure, collectPublish(&collected)),
			},
			&mockClient{
				connected: true,
				close:     closeOK,
				connect:   connectOK,
				publish:   publishFailWith(3, ErrTempBulkFailure, collectPublish(&collected)),
			},
		},
		3,
		100*time.Millisecond,
		100*time.Millisecond,
		1*time.Second,
	)
	testMode(t, mode, events, signals(false), &collected)
}

func TestLoadBalancerTempFlakyFail(t *testing.T) {
	testLoadBalancerTempFlakyFail(t, singleEvent(testEvent))
}

func TestLoadBalancerMultiTempFlakyFail(t *testing.T) {
	testLoadBalancerTempFlakyFail(t, multiEvent(10, testEvent))
}

func testLoadBalancerFlakyInfAttempts(t *testing.T, events []eventInfo) {
	var collected [][]common.MapStr
	mode, _ := NewLoadBalancerMode(
		[]ProtocolClient{
			&mockClient{
				connected: true,
				close:     closeOK,
				connect:   connectOK,
				publish:   publishFailStart(50, collectPublish(&collected)),
			},
			&mockClient{
				connected: true,
				close:     closeOK,
				connect:   connectOK,
				publish:   publishFailStart(50, collectPublish(&collected)),
			},
		},
		0,
		1*time.Nanosecond,
		1*time.Millisecond,
		1*time.Second,
	)
	testMode(t, mode, events, signals(true), &collected)
}

func TestLoadBalancerFlakyInfAttempts(t *testing.T) {
	testLoadBalancerFlakyInfAttempts(t, singleEvent(testEvent))
}

func TestLoadBalancerMultiFlakyInfAttempts(t *testing.T) {
	testLoadBalancerFlakyInfAttempts(t, multiEvent(10, testEvent))
}

func testLoadBalancerTempFlakyInfAttempts(t *testing.T, events []eventInfo) {
	var collected [][]common.MapStr
	mode, _ := NewLoadBalancerMode(
		[]ProtocolClient{
			&mockClient{
				connected: true,
				close:     closeOK,
				connect:   connectOK,
				publish:   publishFailWith(50, ErrTempBulkFailure, collectPublish(&collected)),
			},
			&mockClient{
				connected: true,
				close:     closeOK,
				connect:   connectOK,
				publish:   publishFailWith(50, ErrTempBulkFailure, collectPublish(&collected)),
			},
		},
		0,
		1*time.Nanosecond,
		100*time.Millisecond,
		1*time.Millisecond,
	)
	testMode(t, mode, events, signals(true), &collected)
}

func TestLoadBalancerTempFlakyInfAttempts(t *testing.T) {
	testLoadBalancerTempFlakyInfAttempts(t, singleEvent(testEvent))
}

func TestLoadBalancerMultiTempFlakyInfAttempts(t *testing.T) {
	testLoadBalancerTempFlakyInfAttempts(t, multiEvent(10, testEvent))
}
