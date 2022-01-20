package beater

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

type numberFetcher struct {
	num        int
	stopCalled bool
}

func newNumberFetcher(num int) Fetcher {
	return &numberFetcher{num, false}
}

func (f *numberFetcher) Fetch() ([]interface{}, error) {
	return fetchValue(f.num), nil
}

func fetchValue(num int) []interface{} {
	return []interface{}{num}
}

func (f *numberFetcher) Stop() {
	f.stopCalled = true
}

func registerNFetchers(t *testing.T, d *Data, n int) {
	for i := 0; i < n; i++ {
		key := fmt.Sprint(i)
		err := d.RegisterFetcher(key, newNumberFetcher(i), false)
		if err != nil {
			t.Errorf("failed to register non clashing fetcher with key %s: %v", key, err)
		}

		if _, ok := d.fetcherRegistry[key]; !ok {
			t.Errorf("key %s not found after registration", key)
		}
	}
}

func TestDataRegisterFetcher(t *testing.T) {
	d, err := NewData(context.Background(), duration)
	if err != nil {
		t.Error(err)
	}

	registerNFetchers(t, d, fetcherCount)

	errKey := fmt.Sprint(4)
	err = d.RegisterFetcher(errKey, newNumberFetcher(fetcherCount), false)
	if err == nil {
		t.Errorf("expected error for registering clashing key %s, no error received", errKey)
	}
}

func TestDataRun(t *testing.T) {
	opts := goleak.IgnoreCurrent()

	// Verify no goroutines are leaking. Safest to keep this on top of the function.
	// Go defers are implemented as a LIFO stack. This should be the last one to run.
	defer goleak.VerifyNone(t, opts)

	d, err := NewData(context.Background(), duration)
	if err != nil {
		t.Error(err)
	}

	registerNFetchers(t, d, fetcherCount)
	d.Run()
	defer d.Stop()

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
