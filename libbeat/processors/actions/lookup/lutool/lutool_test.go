package lutool

import (
	"errors"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/stretchr/testify/assert"
)

type funcRunner func(common.MapStr) (common.MapStr, error)

type testRunner struct {
	Runner
	count int
}

type testAction func(t *testing.T, cache *cachedLookup, runner *testRunner) error

func enableLogging(selectors []string) {
	if testing.Verbose() {
		logp.LogInit(logp.LOG_DEBUG, "", false, true, selectors)
	}
}

func TestCacheLookupInitError(t *testing.T) {
	enableLogging([]string{"*"})

	tests := []struct {
		title string
		keys  []string
	}{{
		"Fail no keys",
		nil,
	}}

	for i, test := range tests {
		t.Logf("run (%v): %v", i, test.title)

		exec := false
		_, err := testWithCache(t, test.keys, makeCacheConfig(10),
			func(c *cachedLookup, runner *testRunner) {
				exec = true
			},
		)
		assert.Error(t, err)
		assert.False(t, exec)
	}
}

func TestCachedLookup(t *testing.T) {
	type myint int

	fields1 := common.MapStr{"field": 1}
	fields2 := common.MapStr{"field": 2}
	fields3 := common.MapStr{"field": 3}
	errUPS := errors.New("ups")

	tests := []struct {
		title         string
		keys          []string
		expire        time.Duration
		actions       []testAction
		expectedCalls int
	}{
		{
			"lookup same key twice + success + no expire",
			[]string{"key"},
			100 * time.Second, // do not expire
			[]testAction{
				actSetRunner(fieldsRunner(fields1)),
				actSend(
					common.MapStr{"key": "1", "b": true},
					common.MapStr{"key": "1", "b": true, "fields": fields1}),
				actSend(
					common.MapStr{"key": "1", "b": false},
					common.MapStr{"key": "1", "b": false, "fields": fields1}),
			},
			1,
		},
		{
			"lookup multiple keys twice + success + no expire",
			[]string{"key"},
			100 * time.Second, // do not expire
			[]testAction{
				actSetRunner(fieldsRunner(fields1)),
				actSend(
					common.MapStr{"key": "1", "b": true},
					common.MapStr{"key": "1", "b": true, "fields": fields1}),
				actSetRunner(fieldsRunner(fields2)),
				actSend(
					common.MapStr{"key": "2", "b": true},
					common.MapStr{"key": "2", "b": true, "fields": fields2}),
				actSetRunner(fieldsRunner(fields3)),
				actSend(
					common.MapStr{"key": "1", "b": false},
					common.MapStr{"key": "1", "b": false, "fields": fields1}),
				actSend(
					common.MapStr{"key": "2", "b": false},
					common.MapStr{"key": "2", "b": false, "fields": fields2}),
			},
			2,
		},
		{
			"lookup with error + then success",
			[]string{"key"},
			100 * time.Second, // do not expire
			[]testAction{
				actSetRunner(errRunner(errUPS)),
				actSendUnmodified(common.MapStr{"key": "1"}),

				actWait(50 * time.Millisecond),
				actSendUnmodified(common.MapStr{"key": "1"}),

				actWait(50 * time.Millisecond),
				actSetRunner(fieldsRunner(fields1)),
				actSend(
					common.MapStr{"key": "1", "b": true},
					common.MapStr{"key": "1", "b": true, "fields": fields1}),
			},
			3,
		},
		{
			"do not update event if key fields not available",
			[]string{"key"},
			100 * time.Second, // do not expire
			[]testAction{
				actSetRunner(fieldsRunner(fields1)),
				actSendUnmodified(common.MapStr{"mykey": "1"}),
			},
			0,
		},
		{
			"load expired event",
			[]string{"key"},
			10 * time.Millisecond, // expire very fast
			[]testAction{
				actSetRunner(fieldsRunner(fields1)),
				actSend(
					common.MapStr{"key": "1", "b": true},
					common.MapStr{"key": "1", "b": true, "fields": fields1}),
				actSetRunner(fieldsRunner(fields2)),

				// send another event enforcing janitor to cleanup cache
				actWait(100 * time.Millisecond),
				actSend(
					common.MapStr{"key": "2"},
					common.MapStr{"key": "2", "fields": fields2}),

				// give janitor a chance to drop 'key: 1' and send new event
				actWait(50 * time.Millisecond),
				actSend(
					common.MapStr{"key": "1", "b": false},
					common.MapStr{"key": "1", "b": false, "fields": fields2}),
			},
			3,
		},
		{
			"do not modify event if key is not convertible to string",
			[]string{"key"},
			100 * time.Second, // do not expire
			[]testAction{
				actSetRunner(fieldsRunner(fields1)),
				actSendUnmodified(common.MapStr{"key": myint(2)}),
			},
			0,
		},
		{
			"runner only executed after backoff",
			[]string{"key"},
			100 * time.Second, // do not expire
			[]testAction{
				actSetRunner(errRunner(errUPS)),
				actSendUnmodified(common.MapStr{"key": "1", "b": false}),

				actSetRunner(fieldsRunner(fields1)),
				actRepeat(4, []testAction{
					actSendUnmodified(common.MapStr{"key": "1", "b": false}),
				}),

				actWait(50 * time.Millisecond),
				actSend(
					common.MapStr{"key": "1", "b": false},
					common.MapStr{"key": "1", "b": false, "fields": fields1}),
			},
			2,
		},
	}

	for i, test := range tests {
		t.Logf("run (%v): %v", i, test.title)
		n, err := testWithCache(t, test.keys, makeCacheConfig(test.expire),
			func(c *cachedLookup, runner *testRunner) {
				for _, action := range test.actions {
					if err := action(t, c, runner); err != nil {
						t.Error(err)
						break
					}
				}
			},
		)

		assert.NoError(t, err)
		if test.expectedCalls >= 0 {
			assert.Equal(t, test.expectedCalls, n)
		}
	}
}

func makeCacheConfig(expire time.Duration) CacheConfig {
	return CacheConfig{
		Backoff: BackoffConfig{
			Duration: 10 * time.Millisecond,
			Factor:   2.0,
			Max:      50 * time.Millisecond,
		},
		GCInterval:      10 * time.Millisecond,
		ExpireUnused:    expire,
		FieldsUnderRoot: false,
	}
}

func testWithCache(
	t *testing.T,
	keys []string,
	cg CacheConfig,
	fn func(c *cachedLookup, runner *testRunner),
) (int, error) {
	kb, err := MakeKeyBuilder(keys)
	if len(keys) == 0 {
		assert.Error(t, err)
		return 0, err
	}
	if err != nil {
		t.Error(err)
		return 0, err
	}

	runner := &testRunner{}
	lookup, err := NewCachedLookupTool("test", cg, kb, runner)
	if err != nil {
		return 0, err
	}

	cache := lookup.(*cachedLookup)
	defer cache.cache.Close()

	fn(cache, runner)
	return runner.count, nil
}

func (fn funcRunner) Exec(evt common.MapStr) (common.MapStr, error) {
	return fn(evt)
}

func errRunner(err error) funcRunner {
	return func(_ common.MapStr) (common.MapStr, error) {
		return nil, err
	}
}

func fieldsRunner(fields common.MapStr) funcRunner {
	return func(_ common.MapStr) (common.MapStr, error) {
		return fields.Clone(), nil
	}
}

func (r *testRunner) Exec(evt common.MapStr) (common.MapStr, error) {
	r.count++
	return r.Runner.Exec(evt)
}

func actSetRunner(r Runner) testAction {
	return func(t *testing.T, cache *cachedLookup, runner *testRunner) error {
		runner.Runner = r
		return nil
	}
}

func actSend(event, expected common.MapStr) testAction {
	return func(t *testing.T, cache *cachedLookup, runner *testRunner) error {
		ret, err := cache.Run(event.Clone())
		assert.NoError(t, err)
		if err != nil {
			return err
		}

		b := assert.Equal(t, expected, ret)
		if !b {
			return errors.New("test fail")
		}
		return nil
	}
}

func actSendUnmodified(event common.MapStr) testAction {
	return actSend(event, event)
}

func actWait(d time.Duration) testAction {
	return func(t *testing.T, cache *cachedLookup, runner *testRunner) error {
		time.Sleep(d)
		return nil
	}
}

func actRepeat(N int, actions []testAction) testAction {
	return func(t *testing.T, cache *cachedLookup, runner *testRunner) error {
		for i := 0; i < N; i++ {
			for _, act := range actions {
				if err := act(t, cache, runner); err != nil {
					return err
				}
			}
		}
		return nil
	}
}
