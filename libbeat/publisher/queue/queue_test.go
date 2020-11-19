package queue

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAdjustInternalQueueSize(t *testing.T) {
	t.Run("zero yields default value (main queue size=0)", func(t *testing.T) {
		assert.Equal(t, minInternalQueueSize, AdjustInternalQueueSize(0, 0))
	})
	t.Run("zero yields default value (main queue size=10)", func(t *testing.T) {
		assert.Equal(t, minInternalQueueSize, AdjustInternalQueueSize(0, 10))
	})
	t.Run("can't go below min", func(t *testing.T) {
		assert.Equal(t, minInternalQueueSize, AdjustInternalQueueSize(1, 0))
	})
	t.Run("can set any value within bounds", func(t *testing.T) {
		for q, mainQueue := minInternalQueueSize+1, 4096; q < int(float64(mainQueue)*maxInternalQueueSizeRatio); q += 10 {
			assert.Equal(t, q, AdjustInternalQueueSize(q, mainQueue))
		}
	})
	t.Run("can set any value if no upper bound", func(t *testing.T) {
		for q := minInternalQueueSize + 1; q < math.MaxInt32; q *= 2 {
			assert.Equal(t, q, AdjustInternalQueueSize(q, 0))
		}
	})
	t.Run("can't go above upper bound", func(t *testing.T) {
		mainQueue := 4096
		assert.Equal(t, int(float64(mainQueue)*maxInternalQueueSizeRatio), AdjustInternalQueueSize(mainQueue, mainQueue))
	})
}
