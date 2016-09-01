package modetest

import (
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/outputs"
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

func PublishIgnore([]outputs.Data) ([]outputs.Data, error) {
	return nil, nil
}

func PublishCollect(
	collected *[][]outputs.Data,
) func(data []outputs.Data) ([]outputs.Data, error) {
	mutex := sync.Mutex{}
	return func(data []outputs.Data) ([]outputs.Data, error) {
		mutex.Lock()
		defer mutex.Unlock()

		*collected = append(*collected, data)
		return nil, nil
	}
}

func PublishFailStart(
	n int,
	pub func(data []outputs.Data) ([]outputs.Data, error),
) func(data []outputs.Data) ([]outputs.Data, error) {
	return PublishFailWith(n, errNetTimeout{}, pub)
}

func PublishFailWith(
	n int,
	err error,
	pub func([]outputs.Data) ([]outputs.Data, error),
) func([]outputs.Data) ([]outputs.Data, error) {
	inc := makeCounter(n, err)
	return func(data []outputs.Data) ([]outputs.Data, error) {
		if err := inc(); err != nil {
			return data, err
		}
		return pub(data)
	}
}

func PublishCollectAfterFailStart(
	n int,
	collected *[][]outputs.Data,
) func(data []outputs.Data) ([]outputs.Data, error) {
	return PublishFailStart(n, PublishCollect(collected))
}

func PublishCollectAfterFailStartWith(
	n int,
	err error,
	collected *[][]outputs.Data,
) func(data []outputs.Data) ([]outputs.Data, error) {
	return PublishFailWith(n, err, PublishCollect(collected))
}

func AsyncPublishIgnore(func([]outputs.Data, error), []outputs.Data) error {
	return nil
}

func AsyncPublishCollect(
	collected *[][]outputs.Data,
) func(func([]outputs.Data, error), []outputs.Data) error {
	mutex := sync.Mutex{}
	return func(cb func([]outputs.Data, error), data []outputs.Data) error {
		mutex.Lock()
		defer mutex.Unlock()

		*collected = append(*collected, data)
		cb(nil, nil)
		return nil
	}
}

func AsyncPublishFailStart(
	n int,
	pub func(func([]outputs.Data, error), []outputs.Data) error,
) func(func([]outputs.Data, error), []outputs.Data) error {
	return AsyncPublishFailStartWith(n, errNetTimeout{}, pub)
}

func AsyncPublishFailStartWith(
	n int,
	err error,
	pub func(func([]outputs.Data, error), []outputs.Data) error,
) func(func([]outputs.Data, error), []outputs.Data) error {
	inc := makeCounter(n, err)
	return func(cb func([]outputs.Data, error), data []outputs.Data) error {
		if err := inc(); err != nil {
			return err
		}
		return pub(cb, data)
	}
}

func AsyncPublishCollectAfterFailStart(
	n int,
	collected *[][]outputs.Data,
) func(func([]outputs.Data, error), []outputs.Data) error {
	return AsyncPublishFailStart(n, AsyncPublishCollect(collected))
}

func AsyncPublishCollectAfterFailStartWith(
	n int,
	err error,
	collected *[][]outputs.Data,
) func(func([]outputs.Data, error), []outputs.Data) error {
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
