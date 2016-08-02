package lutool

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExecOnce(t *testing.T) {
	err := errors.New("test")

	tests := []struct {
		title  string
		errors []error
		Worker int
		NRun   int
		NCalls int
		NOK    int
		NFails int
	}{
		{
			"all success",
			nil,
			3,
			5, 1, 15, 0,
		},
		{
			"some fails",
			[]error{err, err},
			3,
			5, 3, 13, 2,
		},
	}

	for i, test := range tests {
		t.Logf("test (%v): %v", i, test.title)

		var exec execOnce

		calls := int32(0)
		oks := int32(0)
		fails := int32(0)

		errors := test.errors

		worker := func() {
			for i := 0; i < test.NRun; i++ {
				err := exec.Do(func() error {
					calls++

					if len(errors) > 0 {
						err := errors[0]
						errors = errors[1:]
						return err
					}
					return nil
				})

				if err != nil {
					atomic.AddInt32(&fails, 1)
				} else {
					atomic.AddInt32(&oks, 1)
				}
			}
		}

		var wg sync.WaitGroup
		for i := 0; i < test.Worker; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				worker()
			}()
		}

		wg.Wait()

		assert.Equal(t, test.NCalls, int(calls))
		assert.Equal(t, test.NOK, int(oks))
		assert.Equal(t, test.NFails, int(fails))
	}
}
