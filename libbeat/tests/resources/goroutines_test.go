package resources

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGoroutinesChecker(t *testing.T) {
	cases := []struct {
		title   string
		test    func()
		timeout time.Duration
		fail    bool
	}{
		{
			title: "no goroutines",
			test:  func() {},
		},
		{
			title: "fast goroutine",
			test:  func() { go func() {}() },
		},
		{
			title: "blocked goroutine",
			test: func() {
				go func() {
					c := make(chan struct{})
					<-c
				}()
			},
			timeout: 500 * time.Millisecond,
			fail:    true,
		},
	}

	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			goroutines := NewGoroutinesChecker()
			if c.timeout > 0 {
				goroutines.FinalizationTimeout = c.timeout
			}
			c.test()
			err := goroutines.check(t)
			if c.fail {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
