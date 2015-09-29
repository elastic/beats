package mode

import (
	"errors"
	"testing"
	"time"

	"github.com/elastic/libbeat/common"
)

func TestSingleSend(t *testing.T) {
	var collected [][]common.MapStr
	mode, _ := NewSingleConnectionMode(
		&mockClient{
			connected: false,
			close:     closeOK,
			connect:   connectOK,
			publish:   collectPublish(&collected),
		},
		3,
		0,
		100*time.Millisecond,
	)
	testMode(t, mode, singleEvent(testEvent), true, &collected)
}

func TestSingleConnectFailConnect(t *testing.T) {
	var collected [][]common.MapStr
	errFail := errors.New("fail connect")
	mode, _ := NewSingleConnectionMode(
		&mockClient{
			connected: false,
			close:     closeOK,
			connect:   failConnect(2, errFail),
			publish:   collectPublish(&collected),
		},
		3,
		0,
		100*time.Millisecond,
	)
	testMode(t, mode, singleEvent(testEvent), true, &collected)
}
