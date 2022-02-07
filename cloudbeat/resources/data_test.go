package resources

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"go.uber.org/goleak"
)

const (
	duration     = 10 * time.Second
	fetcherCount = 10
)

func TestDataRun(t *testing.T) {
	opts := goleak.IgnoreCurrent()

	// Verify no goroutines are leaking. Safest to keep this on top of the function.
	// Go defers are implemented as a LIFO stack. This should be the last one to run.
	defer goleak.VerifyNone(t, opts)

	reg := NewFetcherRegistry()
	registerNFetchers(t, reg, fetcherCount)
	d, err := NewData(duration, reg)
	if err != nil {
		t.Error(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = d.Run(ctx)
	if err != nil {
		return
	}
	defer d.Stop(ctx, cancel)

	o := d.Output()
	state := <-o

	if len(state) < fetcherCount {
		t.Errorf("expected %d keys but got %d", fetcherCount, len(state))
	}

	for i := 0; i < fetcherCount; i++ {
		key := fmt.Sprint(i)

		val, ok := state[key]
		if !ok {
			t.Errorf("expected key %s but not found", key)
		}

		if !reflect.DeepEqual(val, fetchValue(i)) {
			t.Errorf("expected key %s to have value %v but got %v", key, fetchValue(i), val)
		}
	}
}
