package scheduler

import (
	"container/heap"
	"context"
	"sync"
	"time"
)

type TimerTask struct {
	fn    TimerTaskFn
	runAt time.Time
}

type TimerTaskFn func(now time.Time)

type TimerQueue struct {
	th  *TimerHeap
	mtx sync.Mutex
}

func NewTimerQueue() *TimerQueue {
	tq := &TimerQueue{
		th: &TimerHeap{},
	}
	heap.Init(tq.th)

	return tq
}

func (tq *TimerQueue) Push(tt *TimerTask) {
	tq.mtx.Lock()
	defer tq.mtx.Unlock()
	heap.Push(tq.th, tt)
}

func (tq *TimerQueue) PopRunnable() (res []*TimerTask) {
	tq.mtx.Lock()
	defer tq.mtx.Unlock()

	now := time.Now()
	for i := 0; i < tq.th.Len(); i++ {
		peeked := (*tq.th)[0]
		if peeked.runAt.Before(now) {
			popped := heap.Pop(tq.th).(*TimerTask)
			res = append(res, popped)
		} else {
			break
		}
	}

	return res
}

func (tq *TimerQueue) Start(ctx context.Context) {
	// Timer runs every 10ms
	resolution := time.Millisecond * 50
	go func() {
		for {
			select {
			case <-ctx.Done():
				// Stop the timerqueue
				return
			default:
				tq.RunRunnableTasks()
			}

			time.Sleep(resolution)
		}
	}()
}

// RunRunnableTasks runs all tasks that are currently runnable
func (tq *TimerQueue) RunRunnableTasks() {
	runnable := tq.PopRunnable()
	for _, tt := range runnable {
		go tt.fn(time.Now())
	}
}
