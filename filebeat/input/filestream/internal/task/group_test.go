package task

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGroup(t *testing.T) {
	p := NewGroup(10, 10)
	require.NotNil(t, p, "NewGroup returned a nil group, it cannot be nil")

	assert.Equal(t, 10, p.limit)
	assert.Equal(t, 10, p.errsSize)
}

func TestGroup_Go(t *testing.T) {
	t.Run("don't run more than limit goroutines", func(t *testing.T) {
		done := make(chan struct{})
		defer close(done)
		runningCount := atomic.Uint64{}
		blocked := func(_ context.Context) error {
			runningCount.Add(1)
			<-done
			return nil
		}

		want := uint64(2)
		p := NewGroup(want, want)

		err := p.Go(blocked)
		require.NoError(t, err)
		err = p.Go(blocked)
		require.NoError(t, err)
		err = p.Go(blocked)
		require.NoError(t, err)

		assert.Eventually(t,
			func() bool { return uint64(want) == runningCount.Load() },
			time.Second, 100*time.Millisecond)
	})

	t.Run("workloads wait for available worker", func(t *testing.T) {
		runningCount := atomic.Int64{}

		limit := uint64(2)
		p := NewGroup(limit, limit+1)

		done1 := make(chan struct{})
		f1 := func(_ context.Context) error {
			defer t.Log("f1 done")

			runningCount.Add(1)
			defer runningCount.Add(-1)

			t.Log("f1 started")
			<-done1
			return errors.New("f1")
		}

		var f2Finished atomic.Bool
		done2 := make(chan struct{})
		f2 := func(_ context.Context) error {
			defer t.Log("f2 done")

			runningCount.Add(1)

			t.Log("f2 started")
			<-done2

			f2Finished.Store(true)

			runningCount.Add(-1)
			return errors.New("f2")
		}

		var f3Started atomic.Bool
		done3 := make(chan struct{})
		f3 := func(_ context.Context) error {
			defer t.Log("f3 done")

			f3Started.Store(true)
			runningCount.Add(1)

			defer runningCount.Add(-1)
			t.Log("f3 started")
			<-done3
			return errors.New("f3")
		}

		err := p.Go(f1)
		require.NoError(t, err)
		err = p.Go(f2)
		require.NoError(t, err)

		// Wait to ensure f1 and f2 are running, thus there is no workers free.
		assert.Eventually(t,
			func() bool { return int64(2) == runningCount.Load() },
			100*time.Millisecond, time.Millisecond)

		err = p.Go(f3)
		require.NoError(t, err)
		assert.False(t, f3Started.Load())

		close(done2)

		assert.Eventually(t,
			func() bool {
				return f3Started.Load()
			},
			100*time.Millisecond, time.Millisecond)

		// If f3 started, f2 must have finished
		assert.True(t, f2Finished.Load())
		assert.Equal(t, int64(limit), runningCount.Load())

		close(done1)
		close(done3)

		t.Log("waiting the worker pool to finish all workloads")
		err = p.Stop()
		t.Log("worker pool to finished all workloads")

		// Ensure all workloads run
		assert.Contains(t, err.Error(), "f1")
		assert.Contains(t, err.Error(), "f2")
		assert.Contains(t, err.Error(), "f3")
	})

	t.Run("return error if the group is closed", func(t *testing.T) {
		p := NewGroup(1, 1)
		err := p.Stop()
		require.NoError(t, err)

		err = p.Go(func(_ context.Context) error { return nil })
		assert.ErrorIs(t, err, context.Canceled)
	})
}

func TestAddErr(t *testing.T) {
	t.Run("keep most recent errors", func(t *testing.T) {
		p := NewGroup(2, 2)

		p.addErr(errors.New("1"))
		p.addErr(errors.New("2"))
		p.addErr(errors.New("3"))

		err := p.Stop()
		assert.NotContains(t, err.Error(), "1")
		assert.Contains(t, err.Error(), "2")
		assert.Contains(t, err.Error(), "3")
	})

	t.Run("do not add nil errors", func(t *testing.T) {
		p := NewGroup(2, 2)

		p.addErr(nil)
		p.addErr(nil)
		p.addErr(nil)

		for _, err := range p.errs {
			assert.Nil(t, err)
		}
	})
}

func TestPoll_Stop(t *testing.T) {
	t.Run("all workloads return an error", func(t *testing.T) {
		done := make(chan struct{})
		runningCount := atomic.Uint64{}
		wg := sync.WaitGroup{}

		wantErr := errors.New("a error")
		blocked := func(i int) func(context.Context) error {
			return func(_ context.Context) error {
				runningCount.Add(1)
				defer wg.Done()

				select {
				case <-done:
					return fmt.Errorf("[%d]: %w", i, wantErr)
				}
			}
		}

		want := uint64(2)
		p := NewGroup(want, want)

		wg.Add(2)
		err := p.Go(blocked(1))
		require.NoError(t, err)
		err = p.Go(blocked(2))
		require.NoError(t, err)

		close(done)
		wg.Wait()

		err = p.Stop()

		require.Error(t, err)
		assert.Contains(t, err.Error(), wantErr.Error())
		assert.Contains(t, err.Error(), "[2]")
		assert.Contains(t, err.Error(), "[1]")
	})

	t.Run("some workloads return an error", func(t *testing.T) {
		wantErr := errors.New("a error")

		want := uint64(2)
		p := NewGroup(want, want)

		err := p.Go(func(_ context.Context) error { return nil })
		require.NoError(t, err)
		err = p.Go(func(_ context.Context) error { return wantErr })
		require.NoError(t, err)

		time.Sleep(time.Millisecond)

		err = p.Stop()

		require.Error(t, err)
		assert.Contains(t, err.Error(), wantErr.Error())
	})

	t.Run("workload returns no error", func(t *testing.T) {
		done := make(chan struct{})
		runningCount := atomic.Uint64{}
		wg := sync.WaitGroup{}

		bloked := func(i int) func(context.Context) error {
			return func(_ context.Context) error {
				runningCount.Add(1)
				defer wg.Done()

				select {
				case <-done:
					return nil
				}
			}
		}

		want := uint64(2)
		p := NewGroup(want, want)

		wg.Add(2)
		err := p.Go(bloked(1))
		require.NoError(t, err)
		err = p.Go(bloked(2))
		require.NoError(t, err)

		close(done)
		wg.Wait()

		err = p.Stop()

		assert.NoError(t, err)
	})
}
