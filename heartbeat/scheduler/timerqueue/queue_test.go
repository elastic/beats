package timerqueue

import (
	"context"
	"math/rand"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestQueueRunsInOrder(t *testing.T) {
	// Bugs can show up only occasionally
	for i := 0; i < 100; i++ {
		testQueueRunsInOrderOnce(t)
	}
}

func testQueueRunsInOrderOnce(t *testing.T) {
	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()
	tq := NewTimerQueue(ctx)

	// Number of items to test with
	numItems := 10

	// Make a buffered queue for taskResCh so we can easily write to it within this thread.
	taskResCh := make(chan int, numItems)

	// Make a bunch of tasks past their deadline
	var tasks []*TimerTask
	// Start from 1 so we can use the zero value when closing the channel
	for i := 1; i <= numItems; i++ {
		func(i int) {
			schedFor := time.Unix(0, 0).Add(time.Duration(i))
			tasks = append(tasks, NewTimerTask(schedFor, func(now time.Time) {
				taskResCh <- i
				if i == numItems {
					close(taskResCh)
				}
			}))
		}(i)
	}
	// shuffle them so they're out of order
	rand.Shuffle(len(tasks), func(i, j int) { tasks[i], tasks[j] = tasks[j], tasks[i] })

	// insert the randomly ordered events into the queue
	// we use the internal push because pushing and running are in the same threads, so
	// using Push() may result in tasks being executed before all are inserted.
	// This private method is not threadsafe, so is kept private.
	for _, tt := range tasks {
		tq.pushInternal(tt)
	}

	tq.Start()

	var taskResults []int
Reader:
	for {
		select {
		case res := <-taskResCh:
			if res == 0 { // chan closed
				break Reader
			}
			taskResults = append(taskResults, res)
		}
	}

	require.Len(t, taskResults, numItems)
	require.True(t, sort.IntsAreSorted(taskResults), "Results not in order! %v", taskResults)
}

func TestQueueRunsTasksAddedAfterStart(t *testing.T) {
	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()
	tq := NewTimerQueue(ctx)

	tq.Start()

	resCh := make(chan int)
	tq.Push(NewTimerTask(time.Now(), func(now time.Time) {
		resCh <- 1
	}))

	select {
	case r := <-resCh:
		require.Equal(t, 1, r)
	}
}
