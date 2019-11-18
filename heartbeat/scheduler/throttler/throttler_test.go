package throttler

import (
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common/atomic"
	"github.com/stretchr/testify/require"
)

func TestThrottling(t *testing.T) {
	throttler := NewThrottler(5)

	// We should be able to acquire slots without blocking before
	// starting the throttler
	acquiredCount := atomic.NewUint(0)
	for i := 0; i < 5; i++ {
		go func() {
			acquired, release := throttler.AcquireSlot()
			require.True(t, acquired)
			acquiredCount.Inc()
			release()
		}()
	}

	throttler.Start()

	start := time.Now()
	elapsed := time.Duration(0)
	for acquiredCount.Load() < 5 && elapsed < time.Second*10 {
		time.Sleep(time.Millisecond)
		elapsed = time.Now().Sub(start)
	}

	require.Equal(t, acquiredCount.Load(), uint(5))

	throttler.Stop()

	acquired, _ := throttler.AcquireSlot()
	// Acquiring after the throttler is stopped should not work
	require.False(t, acquired)
}
