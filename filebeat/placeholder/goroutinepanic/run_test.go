package goroutinepanic

import (
	"testing"
	"time"
)

func TestWrongPanic(t *testing.T) {
	t.Run("setup failing go-routine", func(t *testing.T) {
		go func() {
			time.Sleep(1 * time.Second)
			t.Fatal("oops")
		}()
	})

	t.Run("false positive failure", func(t *testing.T) {
		time.Sleep(10 * time.Second)
	})
}
