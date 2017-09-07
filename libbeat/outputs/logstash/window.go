package logstash

import (
	"math"
	"sync/atomic"
)

type window struct {
	windowSize      int32
	maxOkWindowSize int // max window size sending was successful for
	maxWindowSize   int
}

func newWindower(start, max int) *window {
	w := &window{}
	w.init(start, max)
	return w
}

func (w *window) init(start, max int) {
	*w = window{
		windowSize:    int32(start),
		maxWindowSize: max,
	}
}

func (w *window) get() int {
	return int(atomic.LoadInt32(&w.windowSize))
}

// Increase window size by factor 1.5 until max window size
// (window size grows exponentially)
// TODO: use duration until ACK to estimate an ok max window size value
func (w *window) tryGrowWindow(batchSize int) {
	windowSize := w.get()

	if windowSize <= batchSize {
		if w.maxOkWindowSize < windowSize {
			w.maxOkWindowSize = windowSize

			newWindowSize := int(math.Ceil(1.5 * float64(windowSize)))

			if windowSize <= batchSize && batchSize < newWindowSize {
				newWindowSize = batchSize
			}
			if newWindowSize > w.maxWindowSize {
				newWindowSize = int(w.maxWindowSize)
			}

			windowSize = newWindowSize
		} else if windowSize < w.maxOkWindowSize {
			windowSize = int(math.Ceil(1.5 * float64(windowSize)))
			if windowSize > w.maxOkWindowSize {
				windowSize = w.maxOkWindowSize
			}
		}

		atomic.StoreInt32(&w.windowSize, int32(windowSize))
	}
}

func (w *window) shrinkWindow() {
	windowSize := w.get()
	orig := windowSize

	windowSize = windowSize / 2
	if windowSize < minWindowSize {
		windowSize = minWindowSize
		if windowSize == orig {
			return
		}
	}

	atomic.StoreInt32(&w.windowSize, int32(windowSize))
}
