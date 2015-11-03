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

func TestLoadBalancerFailSendWithoutActiveConnections(t *testing.T) {
	errFail := errors.New("fail connect")
	mode, _ := NewLoadBalancerMode(
		[]ProtocolClient{
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
	testMode(t, mode, multiEvent(2, testEvent), signals(false), nil)
}

func TestLoadBalancerOKSend(t *testing.T) {
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
	testMode(t, mode, multiEvent(10, testEvent), signals(true), &collected)
}

func TestLoadBalancerFlakyConnectionOkSend(t *testing.T) {
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
	testMode(t, mode, multiEvent(10, testEvent), signals(true), &collected)
}

func TestLoadBalancerTemporayFailure(t *testing.T) {
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
	testMode(t, mode, multiEvent(10, testEvent), signals(true), &collected)
}
