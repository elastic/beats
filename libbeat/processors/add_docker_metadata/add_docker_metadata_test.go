// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//go:build (linux || darwin || windows) && !integration

package add_docker_metadata

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/elastic/beats/v7/libbeat/tests/resources"
	"github.com/elastic/elastic-agent-autodiscover/bus"
	"github.com/elastic/elastic-agent-autodiscover/docker"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/cgroup"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/resolve"
)

type testCGReader struct {
}

func (r testCGReader) ProcessCgroupPaths(pid int) (cgroup.PathList, error) {
	switch pid {
	case 1000:
		return cgroup.PathList{
			V1: map[string]cgroup.ControllerPath{
				"cpu": {ControllerPath: "/docker/8c147fdfab5a2608fe513d10294bf77cb502a231da9725093a155bd25cd1f14b", IsV2: false},
			},
		}, nil
	case 2000:
		return cgroup.PathList{
			V1: map[string]cgroup.ControllerPath{
				"memory": {ControllerPath: "/user.slice", IsV2: false},
			},
		}, nil
	case 3000:
		// Parser error (hopefully this never happens).
		return cgroup.PathList{}, fmt.Errorf("cgroup parse failure")
	default:
		return cgroup.PathList{}, os.ErrNotExist
	}
}

func init() {
	// Stub out the procfs.
	initCgroupPaths = func(_ resolve.Resolver, _ bool) (processors.CGReader, error) {
		return testCGReader{}, nil
	}
}

func TestDefaultProcessorStartup(t *testing.T) {
	// set initCgroupPaths to system non-test defaults
	initCgroupPaths = func(rootfsMountpoint resolve.Resolver, ignoreRootCgroups bool) (processors.CGReader, error) {
		return cgroup.NewReaderOptions(cgroup.ReaderOptions{
			RootfsMountpoint:  rootfsMountpoint,
			IgnoreRootCgroups: ignoreRootCgroups,
		})
	}

	defer func() {
		initCgroupPaths = func(_ resolve.Resolver, _ bool) (processors.CGReader, error) {
			return testCGReader{}, nil
		}
	}()

	rawCfg := defaultConfig()
	cfg, err := config.NewConfigFrom(rawCfg)
	require.NoError(t, err)

	proc, err := buildDockerMetadataProcessor(logp.L(), cfg, docker.NewWatcher)
	require.NoError(t, err)

	unwrapped, _ := proc.(*addDockerMetadata)

	// make sure pid readers have been initialized properly
	_, err = unwrapped.getProcessCgroups(os.Getpid())
	require.NoError(t, err)
}

func TestInitializationNoDocker(t *testing.T) {
	var testConfig = config.NewConfig()
	testConfig.SetString("host", -1, "unix:///var/run42/docker.sock")

	p, err := buildDockerMetadataProcessor(logp.L(), testConfig, docker.NewWatcher)
	assert.NoError(t, err, "initializing add_docker_metadata processor")
	t.Cleanup(func() {
		assert.NoError(t, processors.Close(p), "closing add_docker_metadata processor")
	})

	input := mapstr.M{}
	result, err := p.Run(&beat.Event{Fields: input})
	assert.NoError(t, err, "processing an event")

	assert.Equal(t, mapstr.M{}, result.Fields)
}

func TestInitialization(t *testing.T) {
	var testConfig = config.NewConfig()

	p, err := buildDockerMetadataProcessor(logp.L(), testConfig, MockWatcherFactory(nil, nil))
	assert.NoError(t, err, "initializing add_docker_metadata processor")

	input := mapstr.M{}
	result, err := p.Run(&beat.Event{Fields: input})
	assert.NoError(t, err, "processing an event")

	assert.Equal(t, mapstr.M{}, result.Fields)
}

func TestNoMatch(t *testing.T) {
	testConfig, err := config.NewConfigFrom(map[string]interface{}{
		"match_fields": []string{"foo"},
	})
	assert.NoError(t, err)

	p, err := buildDockerMetadataProcessor(logp.L(), testConfig, MockWatcherFactory(nil, nil))
	assert.NoError(t, err, "initializing add_docker_metadata processor")

	input := mapstr.M{
		"field": "value",
	}
	result, err := p.Run(&beat.Event{Fields: input})
	assert.NoError(t, err, "processing an event")

	assert.Equal(t, mapstr.M{"field": "value"}, result.Fields)
}

func TestMatchNoContainer(t *testing.T) {
	testConfig, err := config.NewConfigFrom(map[string]interface{}{
		"match_fields": []string{"foo"},
	})
	assert.NoError(t, err)

	p, err := buildDockerMetadataProcessor(logp.L(), testConfig, MockWatcherFactory(nil, nil))
	assert.NoError(t, err, "initializing add_docker_metadata processor")

	input := mapstr.M{
		"foo": "garbage",
	}
	result, err := p.Run(&beat.Event{Fields: input})
	assert.NoError(t, err, "processing an event")

	assert.Equal(t, mapstr.M{"foo": "garbage"}, result.Fields)
}

func TestMatchContainer(t *testing.T) {
	testConfig, err := config.NewConfigFrom(map[string]interface{}{
		"match_fields": []string{"foo"},
		"labels.dedot": false,
	})
	assert.NoError(t, err)

	p, err := buildDockerMetadataProcessor(logp.L(), testConfig, MockWatcherFactory(
		map[string]*docker.Container{
			"container_id": {
				ID:    "container_id",
				Image: "image",
				Name:  "name",
				Labels: map[string]string{
					"a.x":   "1",
					"b":     "2",
					"b.foo": "3",
				},
			},
		}, nil))
	assert.NoError(t, err, "initializing add_docker_metadata processor")

	input := mapstr.M{
		"foo": "container_id",
	}
	result, err := p.Run(&beat.Event{Fields: input})
	assert.NoError(t, err, "processing an event")

	assert.EqualValues(t, mapstr.M{
		"container": mapstr.M{
			"id": "container_id",
			"image": mapstr.M{
				"name": "image",
			},
			"labels": mapstr.M{
				"a": mapstr.M{
					"x": "1",
				},
				"b": mapstr.M{
					"value": "2",
					"foo":   "3",
				},
			},
			"name": "name",
		},
		"foo": "container_id",
	}, result.Fields)
}

func TestMatchContainerWithDedot(t *testing.T) {
	testConfig, err := config.NewConfigFrom(map[string]interface{}{
		"match_fields": []string{"foo"},
	})
	assert.NoError(t, err)

	p, err := buildDockerMetadataProcessor(logp.L(), testConfig, MockWatcherFactory(
		map[string]*docker.Container{
			"container_id": {
				ID:    "container_id",
				Image: "image",
				Name:  "name",
				Labels: map[string]string{
					"a.x":   "1",
					"b":     "2",
					"b.foo": "3",
				},
			},
		}, nil))
	assert.NoError(t, err, "initializing add_docker_metadata processor")

	input := mapstr.M{
		"foo": "container_id",
	}
	result, err := p.Run(&beat.Event{Fields: input})
	assert.NoError(t, err, "processing an event")

	assert.EqualValues(t, mapstr.M{
		"container": mapstr.M{
			"id": "container_id",
			"image": mapstr.M{
				"name": "image",
			},
			"labels": mapstr.M{
				"a_x":   "1",
				"b":     "2",
				"b_foo": "3",
			},
			"name": "name",
		},
		"foo": "container_id",
	}, result.Fields)
}

func TestMatchSource(t *testing.T) {
	// Use defaults
	testConfig, err := config.NewConfigFrom(map[string]interface{}{})
	assert.NoError(t, err)

	p, err := buildDockerMetadataProcessor(logp.L(), testConfig, MockWatcherFactory(
		map[string]*docker.Container{
			"8c147fdfab5a2608fe513d10294bf77cb502a231da9725093a155bd25cd1f14b": {
				ID:    "8c147fdfab5a2608fe513d10294bf77cb502a231da9725093a155bd25cd1f14b",
				Image: "image",
				Name:  "name",
				Labels: map[string]string{
					"a": "1",
					"b": "2",
				},
			},
		}, nil))
	assert.NoError(t, err, "initializing add_docker_metadata processor")

	var inputSource string
	switch runtime.GOOS {
	case "windows":
		inputSource = "C:\\ProgramData\\docker\\containers\\FABADA\\foo.log"
	default:
		inputSource = "/var/lib/docker/containers/8c147fdfab5a2608fe513d10294bf77cb502a231da9725093a155bd25cd1f14b/foo.log"
	}
	input := mapstr.M{
		"log": mapstr.M{
			"file": mapstr.M{
				"path": inputSource,
			},
		},
	}

	result, err := p.Run(&beat.Event{Fields: input})
	assert.NoError(t, err, "processing an event")

	assert.EqualValues(t, mapstr.M{
		"container": mapstr.M{
			"id": "8c147fdfab5a2608fe513d10294bf77cb502a231da9725093a155bd25cd1f14b",
			"image": mapstr.M{
				"name": "image",
			},
			"labels": mapstr.M{
				"a": "1",
				"b": "2",
			},
			"name": "name",
		},
		"log": mapstr.M{
			"file": mapstr.M{
				"path": inputSource,
			},
		},
	}, result.Fields)
}

func TestDisableSource(t *testing.T) {
	// Use defaults
	testConfig, err := config.NewConfigFrom(map[string]interface{}{
		"match_source": false,
	})
	assert.NoError(t, err)

	p, err := buildDockerMetadataProcessor(logp.L(), testConfig, MockWatcherFactory(
		map[string]*docker.Container{
			"8c147fdfab5a2608fe513d10294bf77cb502a231da9725093a155bd25cd1f14b": {
				ID:    "8c147fdfab5a2608fe513d10294bf77cb502a231da9725093a155bd25cd1f14b",
				Image: "image",
				Name:  "name",
				Labels: map[string]string{
					"a": "1",
					"b": "2",
				},
			},
		}, nil))
	assert.NoError(t, err, "initializing add_docker_metadata processor")

	input := mapstr.M{
		"source": "/var/lib/docker/containers/8c147fdfab5a2608fe513d10294bf77cb502a231da9725093a155bd25cd1f14b/foo.log",
	}
	result, err := p.Run(&beat.Event{Fields: input})
	assert.NoError(t, err, "processing an event")

	// remains unchanged
	assert.EqualValues(t, input, result.Fields)
}

func TestMatchPIDs(t *testing.T) {
	p, err := buildDockerMetadataProcessor(logp.L(), config.NewConfig(), MockWatcherFactory(
		map[string]*docker.Container{
			"8c147fdfab5a2608fe513d10294bf77cb502a231da9725093a155bd25cd1f14b": {
				ID:    "8c147fdfab5a2608fe513d10294bf77cb502a231da9725093a155bd25cd1f14b",
				Image: "image",
				Name:  "name",
				Labels: map[string]string{
					"a": "1",
					"b": "2",
				},
			},
		},
		nil,
	))
	assert.NoError(t, err, "initializing add_docker_metadata processor")

	dockerMetadata := mapstr.M{
		"container": mapstr.M{
			"id": "8c147fdfab5a2608fe513d10294bf77cb502a231da9725093a155bd25cd1f14b",
			"image": mapstr.M{
				"name": "image",
			},
			"labels": mapstr.M{
				"a": "1",
				"b": "2",
			},
			"name": "name",
		},
	}

	t.Run("pid is not containerized", func(t *testing.T) {
		input := mapstr.M{}
		input.Put("process.pid", 2000)
		input.Put("process.parent.pid", 1000)

		expected := mapstr.M{}
		expected.DeepUpdate(input)

		result, err := p.Run(&beat.Event{Fields: input})
		assert.NoError(t, err, "processing an event")
		assert.EqualValues(t, expected, result.Fields)
	})

	t.Run("pid does not exist", func(t *testing.T) {
		input := mapstr.M{}
		input.Put("process.pid", 9999)

		expected := mapstr.M{}
		expected.DeepUpdate(input)

		result, err := p.Run(&beat.Event{Fields: input})
		assert.NoError(t, err, "processing an event")
		assert.EqualValues(t, expected, result.Fields)
	})

	t.Run("pid is containerized", func(t *testing.T) {
		fields := mapstr.M{}
		fields.Put("process.pid", "1000")

		expected := mapstr.M{}
		expected.DeepUpdate(dockerMetadata)
		expected.DeepUpdate(fields)

		result, err := p.Run(&beat.Event{Fields: fields})
		assert.NoError(t, err, "processing an event")
		assert.EqualValues(t, expected, result.Fields)
	})

	t.Run("pid exited and ppid is containerized", func(t *testing.T) {
		fields := mapstr.M{}
		fields.Put("process.pid", 9999)
		fields.Put("process.parent.pid", 1000)

		expected := mapstr.M{}
		expected.DeepUpdate(dockerMetadata)
		expected.DeepUpdate(fields)

		result, err := p.Run(&beat.Event{Fields: fields})
		assert.NoError(t, err, "processing an event")
		assert.EqualValues(t, expected, result.Fields)
	})

	t.Run("cgroup error", func(t *testing.T) {
		fields := mapstr.M{}
		fields.Put("process.pid", 3000)

		expected := mapstr.M{}
		expected.DeepUpdate(fields)

		result, err := p.Run(&beat.Event{Fields: fields})
		assert.NoError(t, err, "processing an event")
		assert.EqualValues(t, expected, result.Fields)
	})
}

func TestMatchPIDsConcurrent(t *testing.T) {
	containerID := "8c147fdfab5a2608fe513d10294bf77cb502a231da9725093a155bd25cd1f14b"
	p, err := buildDockerMetadataProcessor(logp.NewNopLogger(), config.NewConfig(), MockWatcherFactory(
		map[string]*docker.Container{
			containerID: {
				ID:    containerID,
				Image: "image",
				Name:  "name",
			},
		},
		nil,
	))
	require.NoError(t, err, "initializing add_docker_metadata processor")
	t.Cleanup(func() {
		assert.NoError(t, processors.Close(p), "closing add_docker_metadata processor")
	})

	// Concurrent Run calls on a shared processor must not race on the lazily
	// initialized cgroup cache.
	start := make(chan struct{})
	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			<-start

			fields := mapstr.M{}
			fields.Put("process.pid", 1000)

			result, err := p.Run(&beat.Event{Fields: fields})
			if !assert.NoError(t, err, "processing an event") {
				return
			}
			cid, err := result.Fields.GetValue("container.id")
			assert.NoError(t, err, "getting container.id")
			assert.Equal(t, containerID, cid)
		})
	}
	close(start)
	wg.Wait()
}

// TestCloseBeforeCgroupCacheNoJanitorLeak covers Close running before the lazy cgroup cache is
// initialized. A later cgroupCache call (as a late Run would trigger) must not start a janitor
// goroutine that Close can no longer stop.
func TestCloseBeforeCgroupCacheNoJanitorLeak(t *testing.T) {
	containerID := "8c147fdfab5a2608fe513d10294bf77cb502a231da9725093a155bd25cd1f14b"
	p, err := buildDockerMetadataProcessor(logp.NewNopLogger(), config.NewConfig(), MockWatcherFactory(
		map[string]*docker.Container{
			containerID: {
				ID:    containerID,
				Image: "image",
				Name:  "name",
			},
		},
		nil,
	))
	require.NoError(t, err, "initializing add_docker_metadata processor")
	d := p.(*addDockerMetadata)

	assert.NoError(t, processors.Close(p), "closing processor")
	assert.Nil(t, d.cgroups.Load(), "cgroups should be nil, the cache was never initialized before Close")

	// Stop any janitor a regression would start so it does not affect other tests.
	t.Cleanup(func() {
		if c := d.cgroups.Load(); c != nil {
			c.StopJanitor()
		}
	})

	// Baseline after Close, once all processor goroutines have stopped.
	goroutinesChecker := resources.NewGoroutinesChecker()
	goroutinesChecker.FinalizationTimeout = 2 * time.Second

	d.cgroupCache()

	goroutinesChecker.Check(t)
}

func TestWatcherError(t *testing.T) {
	logger, observedLogs := logptest.NewTestingLoggerWithObserver(t, "")
	testConfig, err := config.NewConfigFrom(map[string]interface{}{
		"match_fields": []string{"foo"},
	})
	assert.NoError(t, err)

	p, err := buildDockerMetadataProcessor(logger, testConfig, MockWatcherFactory(nil, errors.New("mock error")))
	assert.NoError(t, err, "initializing add_docker_metadata processor")
	t.Cleanup(func() {
		assert.NoError(t, processors.Close(p), "closing add_docker_metadata processor")
	})
	assert.Len(t, observedLogs.FilterMessageSnippet("unable to start the docker watcher").TakeAll(), 1)

	input := mapstr.M{
		"field": "value",
	}
	result, err := p.Run(&beat.Event{Fields: input})
	assert.NoError(t, err, "processing an event")
	assert.Equal(t, mapstr.M{"field": "value"}, result.Fields)
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name      string
		cfg       map[string]any
		expectErr bool
	}{
		{
			name: "default",
			cfg:  map[string]any{},
		},
		{
			name: "valid wait timeout",
			cfg: map[string]any{
				"wait_for_metadata":         true,
				"wait_for_metadata_timeout": "20s",
			},
		},
		{
			name: "zero wait timeout",
			cfg: map[string]any{
				"wait_for_metadata":         true,
				"wait_for_metadata_timeout": "0s",
			},
		},
		{
			name: "invalid wait timeout",
			cfg: map[string]any{
				"wait_for_metadata":         true,
				"wait_for_metadata_timeout": "invalid_duration",
			},
			expectErr: true,
		},
		{
			name: "negative wait timeout",
			cfg: map[string]any{
				"wait_for_metadata":         true,
				"wait_for_metadata_timeout": "-1s",
			},
			expectErr: true,
		},
		{
			name: "explicit valid retry period",
			cfg: map[string]any{
				"wait_for_metadata_retry_period": "30s",
			},
		},
		{
			name: "zero retry period",
			cfg: map[string]any{
				"wait_for_metadata_retry_period": "0s",
			},
			expectErr: true,
		},
		{
			name: "negative retry period",
			cfg: map[string]any{
				"wait_for_metadata_retry_period": "-1s",
			},
			expectErr: true,
		},
		{
			name: "invalid retry period duration",
			cfg: map[string]any{
				"wait_for_metadata_retry_period": "not-a-duration",
			},
			expectErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := config.MustNewConfigFrom(test.cfg)
			c := defaultConfig()

			err := cfg.Unpack(&c)
			if test.expectErr {
				require.Error(t, err, "config unpack should fail")
			} else {
				require.NoError(t, err, "config unpack should succeed")
			}
		})
	}
}

func TestInitializationRetriesConnectionToDocker(t *testing.T) {
	var attempts atomic.Int32
	watcherConstructor := func(_ *logp.Logger, host string, tls *docker.TLSConfig, shortID bool) (docker.Watcher, error) {
		attempt := attempts.Add(1)
		if attempt == 1 {
			return nil, errors.New("docker unavailable")
		}

		return &mockWatcher{
			containers: map[string]*docker.Container{
				"container_id": {
					ID:    "container_id",
					Image: "image",
					Name:  "name",
				},
			},
		}, nil
	}

	testConfig := config.MustNewConfigFrom(map[string]any{
		"match_fields":                   []string{"foo"},
		"wait_for_metadata_retry_period": "1ms",
		"wait_for_metadata_timeout":      "1s",
	})

	p, err := buildDockerMetadataProcessor(logp.NewNopLogger(), testConfig, watcherConstructor)
	require.NoError(t, err, "initializing add_docker_metadata processor")
	t.Cleanup(func() {
		assert.NoError(t, processors.Close(p), "closing add_docker_metadata processor")
	})

	assert.Eventually(t, func() bool {
		result, runErr := p.Run(&beat.Event{Fields: mapstr.M{"foo": "container_id"}})
		if runErr != nil {
			return false
		}

		containerID, getErr := result.Fields.GetValue("container.id")
		return getErr == nil && containerID == "container_id"
	}, time.Second, 5*time.Millisecond, "processor should enrich events after retry connects to docker")
	assert.GreaterOrEqual(t, attempts.Load(), int32(2), "watcher constructor should be called more than once")
}

func TestInitializationRetriesUntilTimeout(t *testing.T) {
	var attempts atomic.Int32
	watcherConstructor := func(_ *logp.Logger, host string, tls *docker.TLSConfig, shortID bool) (docker.Watcher, error) {
		attempts.Add(1)
		return nil, errors.New("docker unavailable")
	}

	testConfig := config.MustNewConfigFrom(map[string]any{
		"match_fields":                   []string{"foo"},
		"wait_for_metadata_retry_period": "1ms",
		"wait_for_metadata_timeout":      "10ms",
	})

	p, err := buildDockerMetadataProcessor(logp.NewNopLogger(), testConfig, watcherConstructor)
	require.NoError(t, err, "initializing add_docker_metadata processor")
	t.Cleanup(func() {
		assert.NoError(t, processors.Close(p), "closing add_docker_metadata processor")
	})

	assert.Eventually(t, func() bool {
		return attempts.Load() > 1
	}, time.Second, time.Millisecond, "watcher constructor should be retried until timeout")

	assert.Eventually(t, func() bool {
		previous := attempts.Load()
		time.Sleep(20 * time.Millisecond)
		return previous == attempts.Load()
	}, time.Second, 25*time.Millisecond, "watcher constructor should stop being called after timeout")

	result, runErr := p.Run(&beat.Event{Fields: mapstr.M{"foo": "container_id"}})
	require.NoError(t, runErr, "processing an event")
	assert.Equal(t, mapstr.M{"foo": "container_id"}, result.Fields, "event must remain unchanged without docker connection")
}

func TestInitializationRetriesIndefinitelyWithZeroTimeout(t *testing.T) {
	var attempts atomic.Int32
	watcherConstructor := func(_ *logp.Logger, host string, tls *docker.TLSConfig, shortID bool) (docker.Watcher, error) {
		attempts.Add(1)
		return nil, errors.New("docker unavailable")
	}

	testConfig := config.MustNewConfigFrom(map[string]any{
		"wait_for_metadata_retry_period": "1ms",
		"wait_for_metadata_timeout":      "0s",
	})

	p, err := buildDockerMetadataProcessor(logp.NewNopLogger(), testConfig, watcherConstructor)
	require.NoError(t, err, "initializing add_docker_metadata processor")

	assert.Eventually(t, func() bool {
		return attempts.Load() > 2
	}, time.Second, time.Millisecond, "watcher constructor should keep being retried when timeout is zero")

	require.NoError(t, processors.Close(p), "closing add_docker_metadata processor")
	previous := attempts.Load()
	time.Sleep(20 * time.Millisecond)
	assert.Equal(t, previous, attempts.Load(), "watcher constructor should stop being retried after close")
}

func TestInitializationWaitsForMetadata(t *testing.T) {
	var attempts atomic.Int32
	watcherConstructor := func(_ *logp.Logger, host string, tls *docker.TLSConfig, shortID bool) (docker.Watcher, error) {
		attempt := attempts.Add(1)
		if attempt == 1 {
			return nil, errors.New("docker unavailable")
		}

		return &mockWatcher{
			containers: map[string]*docker.Container{
				"container_id": {
					ID:    "container_id",
					Image: "image",
					Name:  "name",
				},
			},
		}, nil
	}

	testConfig := config.MustNewConfigFrom(map[string]any{
		"match_fields":                   []string{"foo"},
		"wait_for_metadata":              true,
		"wait_for_metadata_retry_period": "1ms",
		"wait_for_metadata_timeout":      "1s",
	})

	p, err := buildDockerMetadataProcessor(logp.NewNopLogger(), testConfig, watcherConstructor)
	require.NoError(t, err, "initializing add_docker_metadata processor")
	t.Cleanup(func() {
		assert.NoError(t, processors.Close(p), "closing add_docker_metadata processor")
	})

	result, runErr := p.Run(&beat.Event{Fields: mapstr.M{"foo": "container_id"}})
	require.NoError(t, runErr, "processing an event")
	containerID, getErr := result.Fields.GetValue("container.id")
	require.NoError(t, getErr, "container metadata should be available immediately after startup")
	assert.Equal(t, "container_id", containerID, "processor should enrich events after synchronous retry connects to docker")
	assert.GreaterOrEqual(t, attempts.Load(), int32(2), "watcher constructor should be called more than once")
}

func TestInitializationWaitForMetadataReturnsErrorOnTimeout(t *testing.T) {
	dockerUnavailable := errors.New("docker unavailable")
	var attempts atomic.Int32
	watcherConstructor := func(_ *logp.Logger, host string, tls *docker.TLSConfig, shortID bool) (docker.Watcher, error) {
		attempts.Add(1)
		return nil, dockerUnavailable
	}

	testConfig := config.MustNewConfigFrom(map[string]any{
		"wait_for_metadata":              true,
		"wait_for_metadata_retry_period": "1ms",
		"wait_for_metadata_timeout":      "10ms",
	})

	p, err := buildDockerMetadataProcessor(logp.NewNopLogger(), testConfig, watcherConstructor)
	require.Error(t, err, "initializing add_docker_metadata processor should fail after timeout")
	assert.ErrorIs(t, err, dockerUnavailable, "error should wrap the last docker connection failure")
	assert.Nil(t, p, "processor should not be returned after wait_for_metadata timeout")
	assert.Greater(t, attempts.Load(), int32(1), "watcher constructor should be retried before timeout")
}

func TestInitializationWaitForMetadataTimeoutIncludesInitialAttempt(t *testing.T) {
	dockerUnavailable := errors.New("docker unavailable")
	var attempts atomic.Int32
	watcherConstructor := func(_ *logp.Logger, host string, tls *docker.TLSConfig, shortID bool) (docker.Watcher, error) {
		attempts.Add(1)
		time.Sleep(20 * time.Millisecond)
		return nil, dockerUnavailable
	}

	testConfig := config.MustNewConfigFrom(map[string]any{
		"wait_for_metadata":              true,
		"wait_for_metadata_retry_period": "1ms",
		"wait_for_metadata_timeout":      "10ms",
	})

	p, err := buildDockerMetadataProcessor(logp.NewNopLogger(), testConfig, watcherConstructor)
	require.Error(t, err, "initializing add_docker_metadata processor should fail after timeout")
	assert.ErrorIs(t, err, dockerUnavailable, "error should wrap the initial docker connection failure")
	assert.Nil(t, p, "processor should not be returned after wait_for_metadata timeout")
	assert.Equal(t, int32(1), attempts.Load(), "timeout should include time spent in the initial attempt")
}

func TestCloseCanBeCalledMultipleTimes(t *testing.T) {
	var stops atomic.Int32
	watcherConstructor := func(_ *logp.Logger, host string, tls *docker.TLSConfig, shortID bool) (docker.Watcher, error) {
		return &mockWatcher{stopCount: &stops}, nil
	}

	p, err := buildDockerMetadataProcessor(logp.NewNopLogger(), config.NewConfig(), watcherConstructor)
	require.NoError(t, err, "initializing add_docker_metadata processor")

	require.NoError(t, processors.Close(p), "first close should succeed")
	require.NoError(t, processors.Close(p), "second close should succeed")
	assert.Equal(t, int32(1), stops.Load(), "watcher should be stopped only once")
}

// Mock container watcher

func MockWatcherFactory(containers map[string]*docker.Container, startErr error) docker.WatcherConstructor {
	if containers == nil {
		containers = make(map[string]*docker.Container)
	}
	return func(_ *logp.Logger, host string, tls *docker.TLSConfig, shortID bool) (docker.Watcher, error) {
		return &mockWatcher{containers: containers, startErr: startErr}, nil
	}
}

type mockWatcher struct {
	containers map[string]*docker.Container
	startErr   error
	stopCount  *atomic.Int32
}

func (m *mockWatcher) Start() error {
	if m.startErr != nil {
		return m.startErr
	}
	return nil
}

func (m *mockWatcher) Stop() {
	if m.stopCount != nil {
		m.stopCount.Add(1)
	}
}

func (m *mockWatcher) Container(ID string) *docker.Container {
	return m.containers[ID]
}

func (m *mockWatcher) Containers() map[string]*docker.Container {
	return m.containers
}

func (m *mockWatcher) ListenStart() bus.Listener {
	return nil
}

func (m *mockWatcher) ListenStop() bus.Listener {
	return nil
}

func BenchmarkAddDockerMetadata(b *testing.B) {
	cfg, err := config.NewConfigFrom(map[string]interface{}{
		"match_fields": []string{"container.id"},
	})
	if err != nil {
		b.Fatal(err)
	}

	p, err := buildDockerMetadataProcessor(logptest.NewTestingLogger(b, ""), cfg, MockWatcherFactory(
		map[string]*docker.Container{
			"abc123": {
				ID:    "abc123def456",
				Image: "myrepo/myimage:latest",
				Name:  "my-container",
				Labels: map[string]string{
					"app":     "myapp",
					"version": "v1.2.3",
					"env":     "production",
				},
			},
		}, nil))
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		event := &beat.Event{
			Fields: mapstr.M{
				"container": mapstr.M{"id": "abc123"},
				"message":   "test log line",
			},
		}
		_, err := p.Run(event)
		if err != nil {
			b.Fatal(err)
		}
	}
}
