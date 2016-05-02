package modetest

import (
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/common"
)

type errNetTimeout struct{}

func (e errNetTimeout) Error() string   { return "errNetTimeout" }
func (e errNetTimeout) Timeout() bool   { return true }
func (e errNetTimeout) Temporary() bool { return false }

func CloseOK() error {
	return nil
}

func ConnectOK(timeout time.Duration) error {
	return nil
}

func ConnectFail(err error) func(time.Duration) error {
	return func(timeout time.Duration) error {
		return err
	}
}

func ConnectFailN(n int, err error) func(time.Duration) error {
	cnt := makeCounter(n, err)
	return func(timeout time.Duration) error {
		return cnt()
	}
}

func PublishIgnore([]common.MapStr) ([]common.MapStr, error) {
	return nil, nil
}

func PublishCollect(
	collected *[][]common.MapStr,
) func(events []common.MapStr) ([]common.MapStr, error) {
	mutex := sync.Mutex{}
	return func(events []common.MapStr) ([]common.MapStr, error) {
		mutex.Lock()
		defer mutex.Unlock()

		*collected = append(*collected, events)
		return nil, nil
	}
}

func PublishFailStart(
	n int,
	pub func(events []common.MapStr) ([]common.MapStr, error),
) func(events []common.MapStr) ([]common.MapStr, error) {
	return PublishFailWith(n, errNetTimeout{}, pub)
}

func PublishFailWith(
	n int,
	err error,
	pub func([]common.MapStr) ([]common.MapStr, error),
) func([]common.MapStr) ([]common.MapStr, error) {
	inc := makeCounter(n, err)
	return func(events []common.MapStr) ([]common.MapStr, error) {
		if err := inc(); err != nil {
			return events, err
		}
		return pub(events)
	}
}

func PublishCollectAfterFailStart(
	n int,
	collected *[][]common.MapStr,
) func(events []common.MapStr) ([]common.MapStr, error) {
	return PublishFailStart(n, PublishCollect(collected))
}

func PublishCollectAfterFailStartWith(
	n int,
	err error,
	collected *[][]common.MapStr,
) func(events []common.MapStr) ([]common.MapStr, error) {
	return PublishFailWith(n, err, PublishCollect(collected))
}

func AsyncPublishIgnore(func([]common.MapStr, error), []common.MapStr) error {
	return nil
}

func AsyncPublishCollect(
	collected *[][]common.MapStr,
) func(func([]common.MapStr, error), []common.MapStr) error {
	mutex := sync.Mutex{}
	return func(cb func([]common.MapStr, error), events []common.MapStr) error {
		mutex.Lock()
		defer mutex.Unlock()

		*collected = append(*collected, events)
		cb(nil, nil)
		return nil
	}
}

func AsyncPublishFailStart(
	n int,
	pub func(func([]common.MapStr, error), []common.MapStr) error,
) func(func([]common.MapStr, error), []common.MapStr) error {
	return AsyncPublishFailStartWith(n, errNetTimeout{}, pub)
}

func AsyncPublishFailStartWith(
	n int,
	err error,
	pub func(func([]common.MapStr, error), []common.MapStr) error,
) func(func([]common.MapStr, error), []common.MapStr) error {
	inc := makeCounter(n, err)
	return func(cb func([]common.MapStr, error), events []common.MapStr) error {
		if err := inc(); err != nil {
			return err
		}
		return pub(cb, events)
	}
}

func AsyncPublishCollectAfterFailStart(
	n int,
	collected *[][]common.MapStr,
) func(func([]common.MapStr, error), []common.MapStr) error {
	return AsyncPublishFailStart(n, AsyncPublishCollect(collected))
}

func AsyncPublishCollectAfterFailStartWith(
	n int,
	err error,
	collected *[][]common.MapStr,
) func(func([]common.MapStr, error), []common.MapStr) error {
	return AsyncPublishFailStartWith(n, err, AsyncPublishCollect(collected))
}

func makeCounter(n int, err error) func() error {
	mutex := sync.Mutex{}
	count := 0

	return func() error {
		mutex.Lock()
		defer mutex.Unlock()

		if count < n {
			count++
			return err
		}
		count = 0
		return nil
	}
}
