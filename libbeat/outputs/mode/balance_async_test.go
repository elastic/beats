package mode

import (
	"errors"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

func TestAsyncLBStartStop(t *testing.T) {
	mode, _ := NewAsyncConnectionMode(
		[]AsyncProtocolClient{},
		false,
		1,
		100*time.Millisecond,
		100*time.Millisecond,
		1*time.Second,
	)
	testMode(t, mode, testNoOpts, nil, nil, nil)
}

func testAsyncLBFailSendWithoutActiveConnection(t *testing.T, events []eventInfo) {
	enableLogging([]string{"*"})

	errFail := errors.New("fail connect")
	mode, _ := NewAsyncConnectionMode(
		[]AsyncProtocolClient{
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
		100*time.Millisecond,
		100*time.Millisecond,
		1*time.Second,
	)
	testMode(t, mode, testNoOpts, events, signals(false), nil)
}

func TestAsyncLBFailSendWithoutActiveConnections(t *testing.T) {
	testAsyncLBFailSendWithoutActiveConnection(t, singleEvent(testEvent))
}

func TestAsyncLBFailSendMultWithoutActiveConnections(t *testing.T) {
	testAsyncLBFailSendWithoutActiveConnection(t, multiEvent(2, testEvent))
}

func testAsyncLBOKSend(t *testing.T, events []eventInfo) {
	enableLogging([]string{"*"})

	var collected [][]common.MapStr
	mode, _ := NewAsyncConnectionMode(
		[]AsyncProtocolClient{
			&mockClient{
				connected:    false,
				close:        closeOK,
				connect:      connectOK,
				asyncPublish: asyncCollectPublish(&collected),
			},
		},
		false,
		2,
		100*time.Millisecond,
		100*time.Millisecond,
		1*time.Second,
	)
	testMode(t, mode, testNoOpts, events, signals(true), &collected)
}

func TestAsyncLBOKSend(t *testing.T) {
	testAsyncLBOKSend(t, singleEvent(testEvent))
}

func TestAsyncLBOKSendMult(t *testing.T) {
	testAsyncLBOKSend(t, multiEvent(10, testEvent))
}

func testAsyncLBFlakyConnectionOkSend(t *testing.T, events []eventInfo) {
	var collected [][]common.MapStr
	mode, _ := NewAsyncConnectionMode(
		[]AsyncProtocolClient{
			&mockClient{
				connected:    true,
				close:        closeOK,
				connect:      connectOK,
				asyncPublish: asyncFailStart(1, asyncCollectPublish(&collected)),
			},
			&mockClient{
				connected:    true,
				close:        closeOK,
				connect:      connectOK,
				asyncPublish: asyncFailStart(1, asyncCollectPublish(&collected)),
			},
		},
		false,
		3,
		100*time.Millisecond,
		100*time.Millisecond,
		1*time.Second,
	)
	testMode(t, mode, testNoOpts, events, signals(true), &collected)
}

func TestAsyncLBFlakyConnectionOkSend(t *testing.T) {
	testAsyncLBFlakyConnectionOkSend(t, singleEvent(testEvent))
}

func TestAsyncLBFlakyConnectionOkSendMult(t *testing.T) {
	testAsyncLBFlakyConnectionOkSend(t, multiEvent(10, testEvent))
}

func testAsyncLBFlakyFail(t *testing.T, events []eventInfo) {
	var collected [][]common.MapStr

	err := errors.New("flaky")
	mode, _ := NewAsyncConnectionMode(
		[]AsyncProtocolClient{
			&mockClient{
				connected:    true,
				close:        closeOK,
				connect:      connectOK,
				asyncPublish: asyncFailWith(3, err, asyncCollectPublish(&collected)),
			},
			&mockClient{
				connected:    true,
				close:        closeOK,
				connect:      connectOK,
				asyncPublish: asyncFailWith(3, err, asyncCollectPublish(&collected)),
			},
		},
		false,
		3,
		100*time.Millisecond,
		100*time.Millisecond,
		1*time.Second,
	)
	testMode(t, mode, testNoOpts, events, signals(false), &collected)
}

func TestAsyncLBFlakyFail(t *testing.T) {
	testAsyncLBFlakyFail(t, singleEvent(testEvent))
}

func TestAsyncLBMultiFlakyFail(t *testing.T) {
	testAsyncLBFlakyFail(t, multiEvent(10, testEvent))
}

func testAsyncLBTemporayFailure(t *testing.T, events []eventInfo) {
	var collected [][]common.MapStr
	mode, _ := NewAsyncConnectionMode(
		[]AsyncProtocolClient{
			&mockClient{
				connected: true,
				close:     closeOK,
				connect:   connectOK,
				asyncPublish: asyncFailWith(1, ErrTempBulkFailure,
					asyncCollectPublish(&collected)),
			},
		},
		false,
		3,
		100*time.Millisecond,
		100*time.Millisecond,
		1*time.Second,
	)
	testMode(t, mode, testNoOpts, events, signals(true), &collected)
}

func TestAsyncLBTemporayFailure(t *testing.T) {
	testAsyncLBTemporayFailure(t, singleEvent(testEvent))
}

func TestAsyncLBTemporayFailureMutlEvents(t *testing.T) {
	testAsyncLBTemporayFailure(t, multiEvent(10, testEvent))
}

func testAsyncLBTempFlakyFail(t *testing.T, events []eventInfo) {
	var collected [][]common.MapStr
	mode, _ := NewAsyncConnectionMode(
		[]AsyncProtocolClient{
			&mockClient{
				connected: true,
				close:     closeOK,
				connect:   connectOK,
				asyncPublish: asyncFailWith(3, ErrTempBulkFailure,
					asyncCollectPublish(&collected)),
			},
			&mockClient{
				connected: true,
				close:     closeOK,
				connect:   connectOK,
				asyncPublish: asyncFailWith(3, ErrTempBulkFailure,
					asyncCollectPublish(&collected)),
			},
		},
		false,
		3,
		100*time.Millisecond,
		100*time.Millisecond,
		1*time.Second,
	)
	testMode(t, mode, testNoOpts, events, signals(false), &collected)
}

func TestAsyncLBTempFlakyFail(t *testing.T) {
	testAsyncLBTempFlakyFail(t, singleEvent(testEvent))
}

func TestAsyncLBMultiTempFlakyFail(t *testing.T) {
	testAsyncLBTempFlakyFail(t, multiEvent(10, testEvent))
}

func testAsyncLBFlakyInfAttempts(t *testing.T, events []eventInfo) {
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"*"})
	}

	var collected [][]common.MapStr
	err := errors.New("flaky")
	mode, _ := NewAsyncConnectionMode(
		[]AsyncProtocolClient{
			&mockClient{
				connected:    true,
				close:        closeOK,
				connect:      connectOK,
				asyncPublish: asyncFailWith(50, err, asyncCollectPublish(&collected)),
			},
			&mockClient{
				connected:    true,
				close:        closeOK,
				connect:      connectOK,
				asyncPublish: asyncFailWith(50, err, asyncCollectPublish(&collected)),
			},
		},
		false,
		0,
		1*time.Nanosecond,
		1*time.Millisecond,
		4*time.Millisecond,
	)
	testMode(t, mode, testNoOpts, events, signals(true), &collected)
}

func TestAsyncLBFlakyInfAttempts(t *testing.T) {
	testAsyncLBFlakyInfAttempts(t, singleEvent(testEvent))
}

func TestAsyncLBMultiFlakyInfAttempts(t *testing.T) {
	testAsyncLBFlakyInfAttempts(t, multiEvent(10, testEvent))
}

func testAsyncLBFlakyInfAttempts2(t *testing.T, events []eventInfo) {
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"*"})
	}

	var collected [][]common.MapStr
	err := errors.New("flaky")
	mode, _ := NewAsyncConnectionMode(
		[]AsyncProtocolClient{
			&mockClient{
				connected:    true,
				close:        closeOK,
				connect:      connectOK,
				asyncPublish: asyncFailStartWith(50, err, asyncCollectPublish(&collected)),
			},
			&mockClient{
				connected:    true,
				close:        closeOK,
				connect:      connectOK,
				asyncPublish: asyncFailStartWith(50, err, asyncCollectPublish(&collected)),
			},
		},
		false,
		0,
		1*time.Nanosecond,
		1*time.Millisecond,
		4*time.Millisecond,
	)
	testMode(t, mode, testNoOpts, events, signals(true), &collected)
}

func TestAsyncLBFlakyInfAttempts2(t *testing.T) {
	testAsyncLBFlakyInfAttempts2(t, singleEvent(testEvent))
}

func TestAsyncLBMultiFlakyInfAttempts2(t *testing.T) {
	testAsyncLBFlakyInfAttempts2(t, multiEvent(10, testEvent))
}

func testAsyncLBFlakyGuaranteed(t *testing.T, events []eventInfo) {
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"*"})
	}

	var collected [][]common.MapStr
	err := errors.New("flaky")
	mode, _ := NewAsyncConnectionMode(
		[]AsyncProtocolClient{
			&mockClient{
				connected:    true,
				close:        closeOK,
				connect:      connectOK,
				asyncPublish: asyncFailWith(50, err, asyncCollectPublish(&collected)),
			},
			&mockClient{
				connected:    true,
				close:        closeOK,
				connect:      connectOK,
				asyncPublish: asyncFailWith(50, err, asyncCollectPublish(&collected)),
			},
		},
		false,
		3,
		1*time.Nanosecond,
		1*time.Millisecond,
		4*time.Millisecond,
	)
	testMode(t, mode, testGuaranteed, events, signals(true), &collected)
}

func TestAsyncLBFlakyGuaranteed(t *testing.T) {
	testAsyncLBFlakyGuaranteed(t, singleEvent(testEvent))
}

func TestAsyncLBMultiFlakyGuaranteed(t *testing.T) {
	testAsyncLBFlakyGuaranteed(t, multiEvent(10, testEvent))
}

func testAsyncLBFlakyGuaranteed2(t *testing.T, events []eventInfo) {
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, []string{"*"})
	}

	var collected [][]common.MapStr
	err := errors.New("flaky")
	mode, _ := NewAsyncConnectionMode(
		[]AsyncProtocolClient{
			&mockClient{
				connected:    true,
				close:        closeOK,
				connect:      connectOK,
				asyncPublish: asyncFailStartWith(50, err, asyncCollectPublish(&collected)),
			},
			&mockClient{
				connected:    true,
				close:        closeOK,
				connect:      connectOK,
				asyncPublish: asyncFailStartWith(50, err, asyncCollectPublish(&collected)),
			},
		},
		false,
		3,
		1*time.Nanosecond,
		1*time.Millisecond,
		4*time.Millisecond,
	)
	testMode(t, mode, testGuaranteed, events, signals(true), &collected)
}

func TestAsyncLBFlakyGuaranteed2(t *testing.T) {
	testAsyncLBFlakyGuaranteed2(t, singleEvent(testEvent))
}

func TestAsyncLBMultiFlakyGuaranteed2(t *testing.T) {
	testAsyncLBFlakyGuaranteed2(t, multiEvent(10, testEvent))
}
