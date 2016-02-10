package mode

import (
	"errors"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common"

	_ "net/http/pprof"
)

func TestLoadBalancerStartStop(t *testing.T) {
	enableLogging([]string{"*"})

	mode, _ := NewLoadBalancerMode(
		[]ProtocolClient{
			&mockClient{
				connected: false,
				close:     closeOK,
				connect:   connectOK,
			},
		},
		1,
		1*time.Millisecond,
		1*time.Millisecond,
		10*time.Millisecond,
	)
	testMode(t, mode, testNoOpts, nil, nil, nil)
}

func testLoadBalancerFailSendWithoutActiveConnections(
	t *testing.T,
	events []eventInfo,
) {
	errFail := errors.New("fail connect")
	mode, _ := NewConnectionMode(
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
		false,
		2,
		1*time.Millisecond,
		1*time.Millisecond,
		10*time.Millisecond,
	)
	testMode(t, mode, testNoOpts, events, signals(false), nil)
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
		1*time.Millisecond,
		1*time.Millisecond,
		10*time.Millisecond,
	)
	testMode(t, mode, testNoOpts, events, signals(true), &collected)
}

func TestLoadBalancerOKSend(t *testing.T) {
	testLoadBalancerOKSend(t, singleEvent(testEvent))
}

func TestLoadBalancerOKSendMult(t *testing.T) {
	testLoadBalancerOKSend(t, multiEvent(10, testEvent))
}

func testLoadBalancerFlakyConnectionOkSend(t *testing.T, events []eventInfo) {
	var collected [][]common.MapStr
	mode, _ := NewConnectionMode(
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
		false,
		3,
		1*time.Millisecond,
		1*time.Millisecond,
		10*time.Millisecond,
	)
	testMode(t, mode, testNoOpts, events, signals(true), &collected)
}

func TestLoadBalancerFlakyConnectionOkSend(t *testing.T) {
	testLoadBalancerFlakyConnectionOkSend(t, singleEvent(testEvent))
}

func TestLoadBalancerFlakyConnectionOkSendMult(t *testing.T) {
	testLoadBalancerFlakyConnectionOkSend(t, multiEvent(10, testEvent))
}

func testLoadBalancerFlakyFail(t *testing.T, events []eventInfo) {
	var collected [][]common.MapStr
	mode, _ := NewConnectionMode(
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
		false,
		3,
		1*time.Millisecond,
		1*time.Millisecond,
		10*time.Millisecond,
	)
	testMode(t, mode, testNoOpts, events, signals(false), &collected)
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
		1*time.Millisecond,
		1*time.Millisecond,
		10*time.Millisecond,
	)
	testMode(t, mode, testNoOpts, events, signals(true), &collected)
}

func TestLoadBalancerTemporayFailure(t *testing.T) {
	testLoadBalancerTemporayFailure(t, singleEvent(testEvent))
}

func TestLoadBalancerTemporayFailureMutlEvents(t *testing.T) {
	testLoadBalancerTemporayFailure(t, multiEvent(10, testEvent))
}

func testLoadBalancerTempFlakyFail(t *testing.T, events []eventInfo) {
	var collected [][]common.MapStr
	mode, _ := NewConnectionMode(
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
		false,
		3,
		1*time.Millisecond,
		1*time.Millisecond,
		10*time.Millisecond,
	)
	testMode(t, mode, testNoOpts, events, signals(false), &collected)
}

func TestLoadBalancerTempFlakyFail(t *testing.T) {
	testLoadBalancerTempFlakyFail(t, singleEvent(testEvent))
}

func TestLoadBalancerMultiTempFlakyFail(t *testing.T) {
	testLoadBalancerTempFlakyFail(t, multiEvent(10, testEvent))
}

func testLoadBalancerFlakyInfAttempts(t *testing.T, events []eventInfo) {
	var collected [][]common.MapStr
	mode, _ := NewConnectionMode(
		[]ProtocolClient{
			&mockClient{
				connected: true,
				close:     closeOK,
				connect:   connectOK,
				publish:   publishFailStart(25, collectPublish(&collected)),
			},
			&mockClient{
				connected: true,
				close:     closeOK,
				connect:   connectOK,
				publish:   publishFailStart(25, collectPublish(&collected)),
			},
		},
		false,
		0,
		1*time.Millisecond,
		1*time.Millisecond,
		10*time.Millisecond,
	)
	testMode(t, mode, testNoOpts, events, signals(true), &collected)
}

func TestLoadBalancerFlakyInfAttempts(t *testing.T) {
	testLoadBalancerFlakyInfAttempts(t, singleEvent(testEvent))
}

func TestLoadBalancerMultiFlakyInfAttempts(t *testing.T) {
	testLoadBalancerFlakyInfAttempts(t, multiEvent(10, testEvent))
}

func testLoadBalancerTempFlakyInfAttempts(t *testing.T, events []eventInfo) {
	var collected [][]common.MapStr
	mode, _ := NewConnectionMode(
		[]ProtocolClient{
			&mockClient{
				connected: true,
				close:     closeOK,
				connect:   connectOK,
				publish:   publishFailWith(25, ErrTempBulkFailure, collectPublish(&collected)),
			},
			&mockClient{
				connected: true,
				close:     closeOK,
				connect:   connectOK,
				publish:   publishFailWith(25, ErrTempBulkFailure, collectPublish(&collected)),
			},
		},
		false,
		0,
		1*time.Millisecond,
		1*time.Millisecond,
		10*time.Millisecond,
	)
	testMode(t, mode, testNoOpts, events, signals(true), &collected)
}

func TestLoadBalancerTempFlakyInfAttempts(t *testing.T) {
	testLoadBalancerTempFlakyInfAttempts(t, singleEvent(testEvent))
}

func TestLoadBalancerMultiTempFlakyInfAttempts(t *testing.T) {
	testLoadBalancerTempFlakyInfAttempts(t, multiEvent(10, testEvent))
}

func testLoadBalancerFlakyGuaranteed(t *testing.T, events []eventInfo) {
	var collected [][]common.MapStr
	mode, _ := NewConnectionMode(
		[]ProtocolClient{
			&mockClient{
				connected: true,
				close:     closeOK,
				connect:   connectOK,
				publish:   publishFailStart(25, collectPublish(&collected)),
			},
			&mockClient{
				connected: true,
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

func TestLoadBalancerFlakyGuaranteed(t *testing.T) {
	testLoadBalancerFlakyGuaranteed(t, singleEvent(testEvent))
}

func TestLoadBalancerMultiFlakyGuaranteed(t *testing.T) {
	testLoadBalancerFlakyGuaranteed(t, multiEvent(10, testEvent))
}

func testLoadBalancerTempFlakyGuaranteed(t *testing.T, events []eventInfo) {
	var collected [][]common.MapStr
	mode, _ := NewConnectionMode(
		[]ProtocolClient{
			&mockClient{
				connected: true,
				close:     closeOK,
				connect:   connectOK,
				publish:   publishFailWith(25, ErrTempBulkFailure, collectPublish(&collected)),
			},
			&mockClient{
				connected: true,
				close:     closeOK,
				connect:   connectOK,
				publish:   publishFailWith(25, ErrTempBulkFailure, collectPublish(&collected)),
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

func TestLoadBalancerTempFlakyGuaranteed(t *testing.T) {
	testLoadBalancerTempFlakyGuaranteed(t, singleEvent(testEvent))
}

func TestLoadBalancerMultiTempFlakyGuaranteed(t *testing.T) {
	testLoadBalancerTempFlakyGuaranteed(t, multiEvent(10, testEvent))
}
