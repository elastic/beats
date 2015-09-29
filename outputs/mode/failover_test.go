package mode

import (
	"errors"
	"testing"
	"time"

	"github.com/elastic/libbeat/common"
)

func TestFailoverSingleSend(t *testing.T) {
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
		100*time.Millisecond,
	)
	testMode(t, mode, singleEvent(testEvent), true, &collected)
}

func TestFailoverFlakyConnections(t *testing.T) {
	errFail := errors.New("fail connect")
	var collected [][]common.MapStr
	mode, _ := NewFailOverConnectionMode(
		[]ProtocolClient{
			&mockClient{
				connected: false,
				close:     closeOK,
				connect:   failConnect(2, errFail),
				publish:   publishTimeoutEvery(1, collectPublish(&collected)),
			},
			&mockClient{
				connected: false,
				close:     closeOK,
				connect:   failConnect(1, errFail),
				publish:   publishTimeoutEvery(2, collectPublish(&collected)),
			},
		},
		3,
		1*time.Millisecond,
		100*time.Millisecond,
	)
	testMode(t, mode, repeatEvent(10, testEvent), true, &collected)
}
