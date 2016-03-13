// +build !integration

package logstash

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShrinkWindowSizeNeverZero(t *testing.T) {
	enableLogging([]string{"logstash"})

	windowSize := 124
	var w window
	w.init(windowSize, defaultConfig.BulkMaxSize)

	w.windowSize = int32(windowSize)
	for i := 0; i < 100; i++ {
		w.shrinkWindow()
	}

	assert.Equal(t, 1, int(w.windowSize))
}

func TestGrowWindowSizeUpToBatchSizes(t *testing.T) {
	batchSize := 114
	windowSize := 1024
	testGrowWindowSize(t, 10, 0, windowSize, batchSize, batchSize)
}

func TestGrowWindowSizeUpToMax(t *testing.T) {
	batchSize := 114
	windowSize := 64
	testGrowWindowSize(t, 10, 0, windowSize, batchSize, windowSize)
}

func TestGrowWindowSizeOf1(t *testing.T) {
	batchSize := 114
	windowSize := 1024
	testGrowWindowSize(t, 1, 0, windowSize, batchSize, batchSize)
}

func TestGrowWindowSizeToMaxOKOnly(t *testing.T) {
	batchSize := 114
	windowSize := 1024
	maxOK := 71
	testGrowWindowSize(t, 1, maxOK, windowSize, batchSize, maxOK)
}

func testGrowWindowSize(t *testing.T,
	initial, maxOK, windowSize, batchSize, expected int,
) {
	enableLogging([]string{"logstash"})
	var w window
	w.init(initial, windowSize)
	w.maxOkWindowSize = maxOK
	for i := 0; i < 100; i++ {
		w.tryGrowWindow(batchSize)
	}

	assert.Equal(t, expected, int(w.windowSize))
	assert.Equal(t, expected, int(w.maxOkWindowSize))
}
